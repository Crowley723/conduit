package certificate

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Crowley723/conduit/internal/config"
	"github.com/Crowley723/conduit/internal/models"
	"github.com/Crowley723/conduit/internal/storage"
	"github.com/Crowley723/conduit/internal/utils"
)

type DatabaseProvider struct {
	storage      storage.Provider
	keyAlgorithm utils.KeyAlgorithm
	subject      *config.CertificateSubject

	// lazily loaded when needed
	caCertMu sync.RWMutex
	caCert   *utils.CertificateData
}

// NewDatabaseProvider creates a new database-backed certificate provider
func NewDatabaseProvider(storage storage.Provider, keyAlgorithm utils.KeyAlgorithm, subject *config.CertificateSubject) *DatabaseProvider {
	return &DatabaseProvider{
		storage:      storage,
		keyAlgorithm: keyAlgorithm,
		subject:      subject,
	}
}

// StartupCheck validates encryption key and ensures CA exists
func (d *DatabaseProvider) StartupCheck(ctx context.Context) error {
	if err := d.storage.ValidateEncryptionKey(ctx); err != nil {
		return fmt.Errorf("encryption key validation failed: %w", err)
	}

	ca, existingAlgorithm, err := d.storage.GetCertificateAuthority(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrCertificateAuthorityNotFound) {
			//TODO: allow ca common_name and expiry to be configurable
			caCert, err := utils.GenerateCA(
				"Homelab Conduit CA",
				3650,
				d.keyAlgorithm,
				d.subject,
			)
			if err != nil {
				return fmt.Errorf("failed to generate CA certificate: %w", err)
			}

			if err := d.storage.InsertCertificateAuthority(ctx, *caCert, d.keyAlgorithm); err != nil {
				return fmt.Errorf("failed to store CA certificate: %w", err)
			}

			d.caCertMu.Lock()
			d.caCert = caCert
			d.caCertMu.Unlock()

			return nil
		}
		return fmt.Errorf("failed to get CA certificate: %w", err)
	}

	if existingAlgorithm != d.keyAlgorithm {
		return fmt.Errorf("%w: configured=%s, existing=%s", storage.ErrKeyAlgorithmMismatch, d.keyAlgorithm, existingAlgorithm)
	}

	d.caCertMu.Lock()
	d.caCert = ca
	d.caCertMu.Unlock()

	return nil
}

// getCA returns the cached CA or loads it from database
func (d *DatabaseProvider) getCA(ctx context.Context) (*utils.CertificateData, error) {
	d.caCertMu.RLock()
	if d.caCert != nil {
		ca := d.caCert
		d.caCertMu.RUnlock()
		return ca, nil
	}
	d.caCertMu.RUnlock()

	d.caCertMu.Lock()
	defer d.caCertMu.Unlock()

	if d.caCert != nil {
		return d.caCert, nil
	}

	ca, _, err := d.storage.GetCertificateAuthority(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get CA certificate: %w", err)
	}

	d.caCert = ca
	return ca, nil
}

func (d *DatabaseProvider) CreateCertificateFromRequest(ctx context.Context, request *models.CertificateRequest) (string, map[string]interface{}, error) {
	ca, err := d.getCA(ctx)
	if err != nil {
		return "", nil, err
	}

	certData, err := utils.GenerateCertificate(request, ca, d.keyAlgorithm, d.subject)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate certificate: %w", err)
	}

	identifier := GenerateCertificateName(request.OwnerSub, request.OwnerIss, time.Now())

	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ca.Certificate.Raw,
	})

	if err := d.storage.InsertIssuedCertificate(ctx, identifier, certData, caCertPEM, d.keyAlgorithm, request.ID, request); err != nil {
		return "", nil, fmt.Errorf("failed to store issued certificate: %w", err)
	}

	metadata := map[string]interface{}{
		"provider":  "database",
		"algorithm": string(d.keyAlgorithm),
	}

	return identifier, metadata, nil
}

func (d *DatabaseProvider) GetCertificateData(ctx context.Context, identifier string) (certPEM, keyPEM, caPEM []byte, err error) {
	return d.storage.GetIssuedCertificateByIdentifier(ctx, identifier)
}

func (d *DatabaseProvider) IsCertificateReady(ctx context.Context, identifier string) (bool, error) {
	_, _, _, err := d.storage.GetIssuedCertificateByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, storage.ErrIssuedCertificateNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *DatabaseProvider) DeleteCertificate(ctx context.Context, identifier string) error {
	return d.storage.DeleteIssuedCertificate(ctx, identifier)
}
