package certificate

import (
	"context"
	"github.com/Crowley723/conduit/internal/models"
)

type DatabaseCertificateProvider struct{}

func (DatabaseCertificateProvider) CreateCertificateFromRequest(ctx context.Context, request *models.CertificateRequest) (string, map[string]interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (DatabaseCertificateProvider) GetCertificateData(ctx context.Context, identifier string) (certPEM, keyPEM, caPEM []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (DatabaseCertificateProvider) IsCertificateReady(ctx context.Context, identifier string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (DatabaseCertificateProvider) DeleteCertificate(ctx context.Context, identifier string) error {
	//TODO implement me
	panic("implement me")
}
