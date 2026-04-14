package handlers

import (
	"encoding/json"
	"github.com/Crowley723/conduit/internal/middlewares"
	"github.com/Crowley723/conduit/internal/models"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ServiceAccount struct {
	Sub          string     `json:"sub"`
	Iss          string     `json:"iss"`
	Name         string     `json:"name"`
	Token        string     `json:"token,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	IsDisabled   bool       `json:"is_disabled"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	Scopes       []string   `json:"scopes,omitempty"`
	CreatedByIss string     `json:"created_by_iss"`
	CreatedBySub string     `json:"created_by_sub"`
	CreatedAt    time.Time  `json:"created_at"`
}

func POSTServiceAccount(ctx *middlewares.AppContext) {
	user, ok := ctx.GetPrincipal().(*models.User)
	if !ok || user == nil {
		ctx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return
	}

	type request struct {
		Name           string   `json:"name"`
		TokenExpiresAt string   `json:"token_expires_at"`
		Scopes         []string `json:"scopes"`
	}

	var req request
	if err := json.NewDecoder(ctx.Request.Body).Decode(&req); err != nil {
		ctx.Logger.Error("failed to decode request body", "error", err)
		ctx.SetJSONError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		ctx.SetJSONError(http.StatusBadRequest, "Service account name is required")
		return
	}

	if len(req.Name) > 255 {
		ctx.SetJSONError(http.StatusBadRequest, "Service account name too long (max 255 characters)")
		return
	}

	if len(req.Scopes) == 0 {
		ctx.SetJSONError(http.StatusBadRequest, "At least one scope is required")
		return
	}

	expiryTime, err := time.Parse(time.RFC3339, req.TokenExpiresAt)
	if err != nil {
		ctx.Logger.Error("failed to parse token expiry time", "error", err)
		ctx.SetJSONError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	if expiryTime.Before(time.Now()) {
		ctx.SetJSONError(http.StatusBadRequest, "Token expiry time must be in the future")
		return
	}

	//TODO: allow configuring max service token lifetime.
	//maxExpiry := time.Now().Add(365 * 24 * time.Hour)
	//if expiryTime.After(maxExpiry) {
	//	ctx.SetJSONError(http.StatusBadRequest, "Token expiry cannot exceed 1 year")
	//	return
	//}

	userScopes := user.GetScopes(ctx.Config)

	if !HasAllScopes(userScopes, req.Scopes) {
		ctx.SetJSONError(http.StatusForbidden, "Cannot grant scopes you don't have")
		return
	}

	token, lookupId, hashedSecret, err := middlewares.GenerateAPIToken()
	if err != nil {
		ctx.Logger.Error("failed to generate API token", "error", err)
		ctx.SetJSONError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	serviceAccount := &models.ServiceAccount{
		Sub:            uuid.New().String(),
		Iss:            ctx.Config.Server.ExternalURL,
		Name:           req.Name,
		LookupId:       lookupId,
		SecretHash:     hashedSecret,
		TokenExpiresAt: expiryTime,
		Scopes:         req.Scopes,
		CreatedByIss:   user.Iss,
		CreatedBySub:   user.Sub,
		CreatedAt:      time.Now(),
	}

	sa, err := ctx.Storage.CreateServiceAccount(ctx, serviceAccount)
	if err != nil {
		ctx.SetJSONError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		ctx.Logger.Error("failed to create service account", "error", err)
		return
	}

	response := ServiceAccount{
		Sub:          sa.Sub,
		Iss:          sa.Iss,
		Name:         sa.Name,
		Token:        token,
		Scopes:       sa.Scopes,
		ExpiresAt:    sa.TokenExpiresAt,
		IsDisabled:   sa.IsDisabled,
		DeletedAt:    sa.DeletedAt,
		CreatedByIss: sa.CreatedByIss,
		CreatedBySub: sa.CreatedBySub,
		CreatedAt:    sa.CreatedAt,
	}

	ctx.WriteJSON(http.StatusCreated, response)
}

func GETServiceAccounts(ctx *middlewares.AppContext) {
	user := ctx.GetPrincipal().(*models.User)

	serviceAccounts, err := ctx.Storage.GetServiceAccountsByCreator(ctx, user.Iss, user.Sub)
	if err != nil {
		ctx.Logger.Error("failed to get service accounts", "error", err)
		ctx.SetJSONError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	response := make([]ServiceAccount, 0, len(serviceAccounts))
	for _, sa := range serviceAccounts {
		response = append(response, ServiceAccount{
			Sub:          sa.Sub,
			Iss:          sa.Iss,
			Name:         sa.Name,
			Scopes:       sa.Scopes,
			ExpiresAt:    sa.TokenExpiresAt,
			IsDisabled:   sa.IsDisabled,
			DeletedAt:    sa.DeletedAt,
			CreatedByIss: sa.CreatedByIss,
			CreatedBySub: sa.CreatedBySub,
			CreatedAt:    sa.CreatedAt,
		})
	}

	ctx.WriteJSON(http.StatusOK, response)
}

func GETServiceAccountWhoami(ctx *middlewares.AppContext) {
	sa, ok := ctx.GetPrincipal().(*models.ServiceAccount)
	if !ok || sa == nil {
		ctx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return
	}

	response := ServiceAccount{
		Sub:          sa.Sub,
		Iss:          sa.Iss,
		Name:         sa.Name,
		Scopes:       sa.Scopes,
		IsDisabled:   sa.IsDisabled,
		DeletedAt:    sa.DeletedAt,
		ExpiresAt:    sa.TokenExpiresAt,
		CreatedByIss: sa.CreatedByIss,
		CreatedBySub: sa.CreatedBySub,
		CreatedAt:    sa.CreatedAt,
	}

	ctx.WriteJSON(http.StatusOK, response)
}

func DELETEServiceAccount(ctx *middlewares.AppContext) {
	user := ctx.GetPrincipal().(*models.User)

	targetIss := ctx.Request.URL.Query().Get("iss")
	targetSub := ctx.Request.URL.Query().Get("sub")

	if targetIss == "" || targetSub == "" {
		ctx.SetJSONError(http.StatusBadRequest, "Missing iss or sub query parameters")
		return
	}

	sa, err := ctx.Storage.GetServiceAccountByID(ctx, targetIss, targetSub)
	if err != nil {
		ctx.Logger.Error("failed to get service account", "error", err)
		ctx.SetJSONError(http.StatusNotFound, "Service account not found")
		return
	}

	if sa.CreatedByIss != user.Iss || sa.CreatedBySub != user.Sub {
		ctx.SetJSONError(http.StatusForbidden, "You can only delete service accounts you created")
		return
	}

	if sa.DeletedAt != nil {
		ctx.SetJSONError(http.StatusGone, "Service account already deleted")
		return
	}

	if err := ctx.Storage.DeleteServiceAccount(ctx, targetIss, targetSub); err != nil {
		ctx.Logger.Error("failed to delete service account", "error", err)
		ctx.SetJSONError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	ctx.WriteJSON(http.StatusOK, map[string]string{"message": "Service account deleted"})
}

func PATCHServiceAccountPause(ctx *middlewares.AppContext) {
	user, ok := ctx.GetPrincipal().(*models.User)
	if !ok || user == nil {
		ctx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return
	}

	targetIss := ctx.Request.URL.Query().Get("iss")
	targetSub := ctx.Request.URL.Query().Get("sub")

	if targetIss == "" || targetSub == "" {
		ctx.SetJSONError(http.StatusBadRequest, "Missing iss or sub query parameters")
		return
	}

	sa, err := ctx.Storage.GetServiceAccountByID(ctx, targetIss, targetSub)
	if err != nil {
		ctx.Logger.Error("failed to get service account", "error", err)
		ctx.SetJSONError(http.StatusNotFound, "Service account not found")
		return
	}

	if sa.CreatedByIss != user.Iss || sa.CreatedBySub != user.Sub {
		ctx.SetJSONError(http.StatusForbidden, "You can only pause service accounts you created")
		return
	}

	if sa.DeletedAt != nil {
		ctx.SetJSONError(http.StatusGone, "Cannot pause a deleted service account")
		return
	}

	if err := ctx.Storage.PauseServiceAccount(ctx, targetIss, targetSub); err != nil {
		ctx.Logger.Error("failed to pause service account", "error", err)
		ctx.SetJSONError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	ctx.WriteJSON(http.StatusOK, map[string]string{"message": "Service account paused"})
}

func PATCHServiceAccountUnpause(ctx *middlewares.AppContext) {
	user, ok := ctx.GetPrincipal().(*models.User)
	if !ok || user == nil {
		ctx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return
	}

	targetIss := ctx.Request.URL.Query().Get("iss")
	targetSub := ctx.Request.URL.Query().Get("sub")

	if targetIss == "" || targetSub == "" {
		ctx.SetJSONError(http.StatusBadRequest, "Missing iss or sub query parameters")
		return
	}

	sa, err := ctx.Storage.GetServiceAccountByID(ctx, targetIss, targetSub)
	if err != nil {
		ctx.Logger.Error("failed to get service account", "error", err)
		ctx.SetJSONError(http.StatusNotFound, "Service account not found")
		return
	}

	if sa.CreatedByIss != user.Iss || sa.CreatedBySub != user.Sub {
		ctx.SetJSONError(http.StatusForbidden, "You can only unpause service accounts you created")
		return
	}

	if sa.DeletedAt != nil {
		ctx.SetJSONError(http.StatusGone, "Cannot unpause a deleted service account")
		return
	}

	if err := ctx.Storage.UnpauseServiceAccount(ctx, targetIss, targetSub); err != nil {
		ctx.Logger.Error("failed to unpause service account", "error", err)
		ctx.SetJSONError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	ctx.WriteJSON(http.StatusOK, map[string]string{"message": "Service account unpaused"})
}

type ScopeInfo struct {
	Scope    string  `json:"scope"`
	Disabled bool    `json:"disabled"`
	Reason   *string `json:"reason,omitempty"`
}

func GETUserScopes(ctx *middlewares.AppContext) {
	user, ok := ctx.GetPrincipal().(*models.User)
	if !ok || user == nil {
		ctx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return
	}

	scopes := user.GetScopes(ctx.Config)
	if scopes == nil {
		scopes = []string{}
	}

	// Convert scopes to ScopeInfo objects with disabled status
	scopeInfos := make([]ScopeInfo, 0, len(scopes))
	firewallDisabledReason := "Firewall management doesn't currently support service accounts"

	for _, scope := range scopes {
		scopeInfo := ScopeInfo{
			Scope:    scope,
			Disabled: false,
		}

		// Disable firewall scopes (all scopes starting with "firewall:")
		if len(scope) >= 9 && scope[:9] == "firewall:" {
			scopeInfo.Disabled = true
			scopeInfo.Reason = &firewallDisabledReason
		}

		scopeInfos = append(scopeInfos, scopeInfo)
	}

	ctx.WriteJSON(http.StatusOK, map[string][]ScopeInfo{"scopes": scopeInfos})
}

func HasAllScopes(userScopes, requestedScopes []string) bool {
	scopeSet := make(map[string]bool, len(userScopes))
	for _, scope := range userScopes {
		scopeSet[scope] = true
	}

	for _, scope := range requestedScopes {
		if !scopeSet[scope] {
			return false
		}
	}

	return true
}
