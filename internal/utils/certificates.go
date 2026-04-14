package utils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"homelab-dashboard/internal/config"
	"homelab-dashboard/internal/models"
	"math/big"
	"time"
)

type CertificateData struct {
	Certificate *x509.Certificate
	PrivateKey  crypto.PrivateKey
}

func GenerateCA(commonName string, validityDays int, algorithm KeyAlgorithm, subject *config.CertificateSubject) (*CertificateData, error) {
	caKey, err := GeneratePrivateKey(algorithm)
	if err != nil {
		return nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	subjectName := pkix.Name{
		CommonName: commonName,
	}

	if subject != nil {
		subjectName.Organization = []string{subject.Organization}
		subjectName.Country = []string{subject.Country}
		subjectName.Locality = []string{subject.Locality}
		subjectName.Province = []string{subject.Province}
	}

	caCertTemplate := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subjectName,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, validityDays),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caCertTemplate, caCertTemplate, publicKey(caKey), caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA certificate: %w", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return &CertificateData{Certificate: caCert, PrivateKey: caKey}, nil
}

func GenerateCertificate(request *models.CertificateRequest, ca *CertificateData, algorithm KeyAlgorithm, subject *config.CertificateSubject) (*CertificateData, error) {
	leafKey, err := GeneratePrivateKey(algorithm)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	subjectName := pkix.Name{
		CommonName: request.CommonName,
	}

	if subject != nil {
		subjectName.Organization = []string{subject.Organization}
		subjectName.Country = []string{subject.Country}
		subjectName.Locality = []string{subject.Locality}
		subjectName.Province = []string{subject.Province}
	}

	if len(request.OrganizationalUnits) > 0 {
		subjectName.OrganizationalUnit = request.OrganizationalUnits
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subjectName,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, request.ValidityDays),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		DNSNames:     request.DNSNames,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, publicKey(leafKey), leafKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return &CertificateData{Certificate: cert, PrivateKey: leafKey}, nil
}

// ParseCertificateDetails extracts details from a PEM-encoded certificate
func ParseCertificateDetails(certPEM []byte) (*models.IssuedCertificateDetails, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return &models.IssuedCertificateDetails{
		SerialNumber: cert.SerialNumber.String(),
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		DNSNames:     cert.DNSNames,
		CommonName:   cert.Subject.CommonName,
		Organization: cert.Subject.Organization,
	}, nil
}

func publicKey(priv crypto.PrivateKey) crypto.PublicKey {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}
