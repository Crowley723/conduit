package certificate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"homelab-dashboard/internal/config"
	"homelab-dashboard/internal/models"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	certmanagerclientset "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesCertificateProvider wraps the Kubernetes and cert-manager clients
type KubernetesCertificateProvider struct {
	ClientSet          *kubernetes.Clientset
	CertManagerClient  *certmanagerclientset.Clientset
	Config             *rest.Config
	Namespace          string
	IssuerName         string
	IssuerKind         string
	CertificateSubject *config.CertificateSubject
	Logger             *slog.Logger
}

// NewKubernetesClient creates a new Kubernetes client based on the configuration
func NewKubernetesClient(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*KubernetesCertificateProvider, error) {
	if cfg.Features == nil || !cfg.Features.MTLSManagement.Enabled {
		return nil, fmt.Errorf("mtls_management is not enabled")
	}

	if cfg.Features == nil {
		return nil, fmt.Errorf("features configuration is nil")

	}

	if cfg.Features.MTLSManagement.Kubernetes == nil {
		return nil, fmt.Errorf("kubernetes configuration is nil")
	}

	k8sCfg := cfg.Features.MTLSManagement.Kubernetes
	issuerCfg := cfg.Features.MTLSManagement.Kubernetes.Issuer
	subjectCfg := cfg.Features.MTLSManagement.CertificateSubject

	if issuerCfg == nil {
		return nil, fmt.Errorf("certificate issuer configuration is missing")
	}

	if k8sCfg == nil || k8sCfg.Namespace == "" {
		return nil, fmt.Errorf("kubernetes namespace configuration is missing")
	}

	var restConfig *rest.Config
	var err error

	if k8sCfg.InCluster {
		logger.Info("using in-cluster Kubernetes configuration")
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		kubeconfig := k8sCfg.Kubeconfig
		if kubeconfig == "" {
			if home := homeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}

		logger.Debug("Using Kubeconfig File", "path", kubeconfig)
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	certManagerClient, err := certmanagerclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert-manager clientset: %w", err)
	}

	_, err = clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return nil, fmt.Errorf("failed to verify kubernetes connection: %w", err)
	}

	return &KubernetesCertificateProvider{
		ClientSet:          clientset,
		CertManagerClient:  certManagerClient,
		Config:             restConfig,
		Namespace:          k8sCfg.Namespace,
		IssuerName:         issuerCfg.Name,
		IssuerKind:         issuerCfg.Kind,
		CertificateSubject: subjectCfg,
		Logger:             logger,
	}, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

// CreateCertificateFromRequest creates a cert-manager Certificate resource from a CertificateRequest
func (c *KubernetesCertificateProvider) CreateCertificateFromRequest(ctx context.Context, request *models.CertificateRequest) (string, map[string]interface{}, error) {
	certName := GenerateCertificateName(request.OwnerSub, request.OwnerIss, request.RequestedAt)
	secretName := fmt.Sprintf("%s-tls", certName)

	duration := time.Duration(request.ValidityDays) * 24 * time.Hour

	issuerRef := cmmeta.IssuerReference{
		Name: c.IssuerName,
		Kind: c.IssuerKind,
	}

	// If using Issuer (not ClusterIssuer), set the group
	if c.IssuerKind == "Issuer" {
		group := "cert-manager.io"
		issuerRef.Group = group
	}

	subject := &certmanagerv1.X509Subject{}

	if len(request.OrganizationalUnits) > 0 {
		subject.OrganizationalUnits = request.OrganizationalUnits
	}

	if c.CertificateSubject != nil && c.CertificateSubject.Organization != "" {
		subject.Organizations = []string{c.CertificateSubject.Organization}
	}

	if c.CertificateSubject != nil && c.CertificateSubject.Country != "" {
		subject.Countries = []string{c.CertificateSubject.Country}
	}

	if c.CertificateSubject != nil && c.CertificateSubject.Province != "" {
		subject.Provinces = []string{c.CertificateSubject.Province}
	}

	if c.CertificateSubject != nil && c.CertificateSubject.Locality != "" {
		subject.Localities = []string{c.CertificateSubject.Locality}
	}

	subject.SerialNumber = fmt.Sprintf("%s@%s", request.OwnerSub, removeURLSchemeAndSlashes(request.OwnerIss))

	cert := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: c.Namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByConduit,
				LabelOwnerSub:  sanitizeLabelValue(request.OwnerSub),
				LabelOwnerIss:  sanitizeLabelValue(removeURLSchemeAndSlashes(request.OwnerIss)),
				LabelRequestID: fmt.Sprintf("%d", request.ID),
			},
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName: secretName,
			Duration: &metav1.Duration{
				Duration: duration,
			},
			IssuerRef:  issuerRef,
			CommonName: request.CommonName,
			DNSNames:   request.DNSNames,
			Subject:    subject,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},
	}

	c.Logger.Debug("creating certificate",
		"name", certName,
		"namespace", c.Namespace,
		"commonName", request.CommonName,
		"requestID", request.ID)

	created, err := c.CertManagerClient.CertmanagerV1().Certificates(c.Namespace).Create(ctx, cert, metav1.CreateOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	c.Logger.Debug("Certificate Created Successfully",
		"name", created.Name,
		"namespace", created.Namespace)

	// Return metadata about the Kubernetes resources
	metadata := map[string]interface{}{
		"namespace":   created.Namespace,
		"secret_name": created.Spec.SecretName,
	}

	return certName, metadata, nil
}

// GetCertificate retrieves a Certificate resource
func (c *KubernetesCertificateProvider) GetCertificate(ctx context.Context, name string) (*certmanagerv1.Certificate, error) {
	cert, err := c.CertManagerClient.CertmanagerV1().Certificates(c.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("certificate not found: %s/%s", c.Namespace, name)
		}
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}
	return cert, nil
}

func (c *KubernetesCertificateProvider) GetCertificateData(ctx context.Context, name string) (certPEM, keyPEM, caPEM []byte, err error) {
	cert, err := c.GetCertificate(ctx, name)
	if err != nil {
		return nil, nil, nil, err
	}

	isReady := false
	for _, condition := range cert.Status.Conditions {
		if condition.Type == certmanagerv1.CertificateConditionReady {
			isReady = condition.Status == cmmeta.ConditionTrue
			break
		}
	}

	if !isReady {
		return nil, nil, nil, fmt.Errorf("certificate is not ready yet")
	}

	secret, err := c.getCertificateSecret(ctx, cert.Spec.SecretName)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get certificate data: %w", err)
	}

	certPEM = secret.Data["tls.crt"]
	keyPEM = secret.Data["tls.key"]
	caPEM = secret.Data["ca.crt"]

	if len(certPEM) == 0 {
		return nil, nil, nil, fmt.Errorf("failed to get certificate data: certificate secret missing tls.crt")
	}
	if len(keyPEM) == 0 {
		return nil, nil, nil, fmt.Errorf("failed to get certificate data: certificate secret missing tls.key")
	}

	return certPEM, keyPEM, caPEM, nil

}

// IsCertificateReady checks if a Certificate is ready
func (c *KubernetesCertificateProvider) IsCertificateReady(ctx context.Context, name string) (bool, error) {
	cert, err := c.GetCertificate(ctx, name)
	if err != nil {
		return false, err
	}

	for _, condition := range cert.Status.Conditions {
		if condition.Type == certmanagerv1.CertificateConditionReady {
			return condition.Status == cmmeta.ConditionTrue, nil
		}
	}

	return false, nil
}

// getCertificateSecret retrieves the Secret containing the issued certificate
func (c *KubernetesCertificateProvider) getCertificateSecret(ctx context.Context, secretName string) (*corev1.Secret, error) {
	secret, err := c.ClientSet.CoreV1().Secrets(c.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("certificate secret not found: %s/%s", c.Namespace, secretName)
		}
		return nil, fmt.Errorf("failed to get certificate secret: %w", err)
	}
	return secret, nil
}

// DeleteCertificate deletes a Certificate resource
func (c *KubernetesCertificateProvider) DeleteCertificate(ctx context.Context, name string) error {
	c.Logger.DebugContext(ctx, "deleting certificate", "name", name, "namespace", c.Namespace)

	err := c.CertManagerClient.CertmanagerV1().Certificates(c.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.Logger.WarnContext(ctx, "failed to delete certificate: certificate already deleted", "name", name, "namespace", c.Namespace)
			return nil
		}
		return fmt.Errorf("failed to delete certificate: %w", err)
	}

	c.Logger.Info("certificate deleted successfully", "name", name, "namespace", c.Namespace)
	return nil
}

// removeURLSchemeAndSlashes removes https://, http://, and all slashes from a string
func removeURLSchemeAndSlashes(value string) string {
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")

	value = strings.ReplaceAll(value, "/", "")

	return value
}

// sanitizeLabelValue ensures the label value meets Kubernetes requirements
// Label values must be 63 characters or less and match regex: [a-z0-9A-Z]([-_.a-z0-9A-Z]*[a-z0-9A-Z])?
func sanitizeLabelValue(value string) string {
	// Remove URL scheme and slashes first
	value = removeURLSchemeAndSlashes(value)

	// Remove any characters not in the allowed set
	var result strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.' {
			result.WriteRune(char)
		}
	}

	sanitized := result.String()

	// Ensure it starts and ends with alphanumeric
	sanitized = strings.TrimLeft(sanitized, "-_.")
	sanitized = strings.TrimRight(sanitized, "-_.")

	// Truncate to 63 characters
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
		sanitized = strings.TrimRight(sanitized, "-_.")
	}

	if sanitized == "" {
		hash := sha256.Sum256([]byte(value))
		sanitized = hex.EncodeToString(hash[:8])
	}

	return sanitized
}
