package handlers

import (
	"net/http"

	"github.com/Crowley723/conduit/internal/middlewares"
	"github.com/Crowley723/conduit/internal/models"
)

type ConfigResponse struct {
	MTLS     MTLSConfigResponse     `json:"mtls,omitempty"`
	Firewall FirewallConfigResponse `json:"firewall,omitempty"`
}

type MTLSConfigResponse struct {
	Enabled bool `json:"enabled"`
}

type FirewallConfigResponse struct {
	Enabled bool `json:"enabled"`
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
		ctx.WriteJSON(http.StatusUnauthorized, response)
		return
	}

	config := &ConfigResponse{}

	if ctx.Config.Features != nil && ctx.Config.Features.MTLSManagement.Enabled {
		config.MTLS = MTLSConfigResponse{
			Enabled: true,
		}
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
