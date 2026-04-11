package handlers

import (
	"net/http"

	"github.com/Crowley723/conduit/internal/middlewares"
	"github.com/Crowley723/conduit/internal/models"
)

type ConfigResponse struct {
	MTLS MTLSConfigResponse `json:"mtls,omitempty"`
}

type MTLSConfigResponse struct {
	Enabled            bool                        `json:"enabled"`
	CertificateSubject *CertificateSubjectResponse `json:"certificate_subject,omitempty"`
	KeyAlgorithm       string                      `json:"key_algorithm,omitempty"`
	ProviderType       string                      `json:"provider_type,omitempty"`
}

type CertificateSubjectResponse struct {
	Organization string `json:"organization,omitempty"`
	Country      string `json:"country,omitempty"`
	Locality     string `json:"locality,omitempty"`
	Province     string `json:"province,omitempty"`
}

type AuthStatusResponse struct {
	Authenticated bool            `json:"authenticated"`
	User          *models.User    `json:"user,omitempty"`
	Config        *ConfigResponse `json:"config,omitempty"`
}

func GETAuthStatusHandler(ctx *middlewares.AppContext) {
	response := AuthStatusResponse{
		Authenticated: false,
	}

	if !ctx.SessionManager.IsUserAuthenticated(ctx) {
		ctx.WriteJSON(http.StatusOK, response)
		return
	}

	config := &ConfigResponse{}

	if ctx.Config.MTLS.Enabled {
		mtlsConfig := MTLSConfigResponse{
			Enabled: true,
		}

		if ctx.Config.MTLS.CertificateSubject != nil {
			mtlsConfig.CertificateSubject = &CertificateSubjectResponse{
				Organization: ctx.Config.MTLS.CertificateSubject.Organization,
				Country:      ctx.Config.MTLS.CertificateSubject.Country,
				Locality:     ctx.Config.MTLS.CertificateSubject.Locality,
				Province:     ctx.Config.MTLS.CertificateSubject.Province,
			}
		}

		if ctx.Config.MTLS.Kubernetes != nil && ctx.Config.MTLS.Kubernetes.Enabled {
			mtlsConfig.ProviderType = "kubernetes"
		} else if ctx.Config.MTLS.Database != nil && ctx.Config.MTLS.Database.Enabled {
			mtlsConfig.ProviderType = "database"
			mtlsConfig.KeyAlgorithm = ctx.Config.MTLS.Database.KeyAlgorithm
		}

		config.MTLS = mtlsConfig
	}

	response.Config = config

	user, ok := ctx.SessionManager.GetAuthenticatedUser(ctx)
	if user == nil {
		ctx.WriteJSON(http.StatusUnauthorized, response)
		return
	}

	if ok {
		response.Authenticated = true
		response.User = user
		ctx.WriteJSON(http.StatusOK, response)
		return
	}

	ctx.WriteJSON(http.StatusUnauthorized, response)
}
