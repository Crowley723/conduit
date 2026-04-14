package certificate

import (
	"context"
	"github.com/Crowley723/conduit/internal/models"
)

type Provider interface {
	// CreateCertificateFromRequest creates a certificate and returns an identifier and metadata
	CreateCertificateFromRequest(ctx context.Context, request *models.CertificateRequest) (identifier string, metadata map[string]interface{}, err error)

	// GetCertificateData retrieves certificate data by identifier
	GetCertificateData(ctx context.Context, identifier string) (certPEM, keyPEM, caPEM []byte, err error)

	// IsCertificateReady checks if a certificate is ready by identifier
	IsCertificateReady(ctx context.Context, identifier string) (ready bool, err error)

	// DeleteCertificate deletes a certificate by identifier
	DeleteCertificate(ctx context.Context, identifier string) error
}
