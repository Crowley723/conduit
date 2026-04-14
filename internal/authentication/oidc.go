package authentication

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/Crowley723/conduit/internal/config"
	"github.com/Crowley723/conduit/internal/middlewares"
	"github.com/Crowley723/conduit/internal/models"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// NewRealOIDCProvider creates a new instance of the real OIDC provider with initialized config
func NewRealOIDCProvider(ctx context.Context, cfg config.OIDCConfig) (middlewares.OIDCProvider, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
		RedirectURL:  cfg.RedirectURI,
	}

	return &RealOIDCProvider{
		provider:     provider,
		oauth2Config: oauth2Config,
	}, nil
}

type RealOIDCProvider struct {
	provider     *oidc.Provider
	oauth2Config *oauth2.Config
}

func (r *RealOIDCProvider) GetProvider() *oidc.Provider {
	return r.provider
}

func (r *RealOIDCProvider) GetOAuth2Config() *oauth2.Config {
	return r.oauth2Config
}

func (r *RealOIDCProvider) GenerateRandString(bytes int) string {
	if bytes <= 0 {
		bytes = 32
	}

	b := make([]byte, bytes)
	_, _ = rand.Read(b)

	return base64.URLEncoding.EncodeToString(b)
}

func (r *RealOIDCProvider) GenerateCodeVerifier() (string, string) {
	b := make([]byte, 56)
	_, _ = rand.Read(b)

	codeVerifier := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b)
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
	return codeVerifier, codeChallenge
}

func (r *RealOIDCProvider) StartLogin(ctx *middlewares.AppContext) (string, error) {
	state := r.GenerateRandString(32)
	nonce := r.GenerateRandString(32)
	codeVerifier, codeChallenge := r.GenerateCodeVerifier()

	ctx.SessionManager.SetOauthNonce(ctx, nonce)
	ctx.SessionManager.SetOauthState(ctx, state)
	ctx.SessionManager.SetOauthCodeVerifier(ctx, codeVerifier)

	authURL := ctx.OIDCProvider.GetOAuth2Config().AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("prompt", "login"),
		oauth2.SetAuthURLParam("response_type", "code"),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	return authURL, nil
}

func (r *RealOIDCProvider) HandleCallback(ctx *middlewares.AppContext) (*oidc.IDToken, *models.User, error) {
	if errorParam := ctx.Request.URL.Query().Get("error"); errorParam != "" {
		errorDescription := ctx.Request.URL.Query().Get("error_description")
		errorURI := ctx.Request.URL.Query().Get("error_uri")
		state := ctx.Request.URL.Query().Get("state")

		errorURL := fmt.Sprintf("/error?error=%s", url.QueryEscape(errorParam))
		if errorDescription != "" {
			errorURL += "&error_description=" + url.QueryEscape(errorDescription)
		}
		if errorURI != "" {
			errorURL += "&error_uri=" + url.QueryEscape(errorURI)
		}
		if state != "" {
			errorURL += "&state=" + url.QueryEscape(state)
		}

		return nil, nil, &OIDCError{RedirectURL: errorURL, Message: errorParam}
	}

	storedState := ctx.SessionManager.GetOauthState(ctx)
	if storedState == "" {
		return nil, nil, &OIDCError{
			RedirectURL: "/error?error=invalid_request&error_description=" + url.QueryEscape("No oauth state found in session"),
			Message:     "no oauth state found in session",
		}
	}

	receivedState := ctx.Request.URL.Query().Get("state")
	if receivedState != storedState {
		return nil, nil, &OIDCError{
			RedirectURL: "/error?error=invalid_request&error_description=" + url.QueryEscape("Invalid state parameter"),
			Message:     "invalid state parameter",
		}
	}

	ctx.SessionManager.ClearOauthState(ctx)

	code := ctx.Request.URL.Query().Get("code")
	if code == "" {
		return nil, nil, &OIDCError{
			RedirectURL: "/error?error=invalid_request&error_description=" + url.QueryEscape("No authorization code received"),
			Message:     "no authorization code received",
		}
	}

	verifierCode := ctx.SessionManager.GetOauthCodeVerifier(ctx)
	ctx.SessionManager.ClearOauthCodeVerifier(ctx)

	token, err := ctx.OIDCProvider.GetOAuth2Config().Exchange(ctx.Request.Context(), code, oauth2.VerifierOption(verifierCode))
	if err != nil {
		return nil, nil, &OIDCError{
			RedirectURL: "/error?error=invalid_grant&error_description=" + url.QueryEscape("Failed to exchange code for token"),
			Message:     fmt.Sprintf("failed to exchange code for token: %v", err),
		}
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, nil, &OIDCError{
			RedirectURL: "/error?error=invalid_token&error_description=" + url.QueryEscape("No id_token found in oauth2 token"),
			Message:     "no id_token found in oauth2 token",
		}
	}

	verifier := ctx.OIDCProvider.GetProvider().Verifier(&oidc.Config{ClientID: ctx.OIDCProvider.GetOAuth2Config().ClientID})

	idToken, err := verifier.Verify(ctx.Request.Context(), rawIDToken)
	if err != nil {
		return nil, nil, &OIDCError{
			RedirectURL: "/error?error=invalid_token&error_description=" + url.QueryEscape("Failed to verify ID Token"),
			Message:     fmt.Sprintf("failed to verify ID Token: %v", err),
		}
	}

	user, nonce, err := extractUserClaimsFromToken(idToken)
	if err != nil {
		return nil, nil, &OIDCError{
			RedirectURL: "/error?error=server_error&error_description=" + url.QueryEscape("Failed to extract user from ID Token"),
			Message:     fmt.Sprintf("failed to extract user from ID Token: %v", err),
		}
	}

	if nonce != ctx.SessionManager.GetOauthNonce(ctx) {
		return nil, nil, &OIDCError{
			RedirectURL: "/error?error=server_error&error_description=" + url.QueryEscape("Invalid Nonce"),
			Message:     fmt.Sprintf("nonce in ID Token is invalid"),
		}
	}

	ctx.SessionManager.ClearOauthNonce(ctx)

	enhancedUser, err := fetchUserInfo(ctx, token, user)
	if err != nil {
		ctx.Logger.Warn("Failed to fetch user info, using ID token data only", "error", err)
		enhancedUser = user
	}

	return idToken, enhancedUser, nil

}

// fetchUserInfo retrieves additional user information from the UserInfo endpoint
func fetchUserInfo(ctx *middlewares.AppContext, token *oauth2.Token, baseUser *models.User) (*models.User, error) {
	userInfo, err := ctx.OIDCProvider.GetProvider().UserInfo(context.Background(), oauth2.StaticTokenSource(token))
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	enhancedUser := &models.User{
		Sub: baseUser.Sub,
		Iss: baseUser.Iss,
	}

	var claims struct {
		Username    string   `json:"preferred_username"`
		Name        string   `json:"name"`
		DisplayName string   `json:"display_name"`
		Email       string   `json:"email"`
		Groups      []string `json:"groups"`
	}

	// Extract claims from UserInfo
	if err := userInfo.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse user info claims: %w", err)
	}

	enhancedUser.Username = getPreferredValue(claims.Username, baseUser.Username)
	enhancedUser.DisplayName = getPreferredValue(claims.DisplayName, claims.Name, baseUser.DisplayName)
	enhancedUser.Email = getPreferredValue(claims.Email, baseUser.Email)
	enhancedUser.Groups = claims.Groups

	if len(enhancedUser.Groups) == 0 {
		enhancedUser.Groups = baseUser.Groups
	}

	return enhancedUser, nil
}

// getPreferredValue returns the first non-empty string from the provided values
func getPreferredValue(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
