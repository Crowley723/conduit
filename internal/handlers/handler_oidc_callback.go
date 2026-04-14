package handlers

import (
	"errors"
	"fmt"
	"github.com/Crowley723/conduit/internal/authentication"
	"github.com/Crowley723/conduit/internal/middlewares"
	"net/http"
	"net/url"
)

func GETCallbackHandler(ctx *middlewares.AppContext) {
	if errorParam := ctx.Request.URL.Query().Get("error"); errorParam != "" {
		errorDesc := ctx.Request.URL.Query().Get("error_description")
		errorURI := ctx.Request.URL.Query().Get("error_uri")
		state := ctx.Request.URL.Query().Get("state")

		ctx.Logger.Warn("OIDC callback error", "error", errorParam, "description", errorDesc)

		errorURL := "/error?error=" + url.QueryEscape(errorParam)
		if errorDesc != "" {
			errorURL += "&error_description=" + url.QueryEscape(errorDesc)
		}
		if errorURI != "" {
			errorURL += "&error_uri=" + url.QueryEscape(errorURI)
		}
		if state != "" {
			errorURL += "&state=" + url.QueryEscape(state)
		}

		ctx.Redirect(errorURL, http.StatusFound)
		return
	}

	idToken, user, err := ctx.OIDCProvider.HandleCallback(ctx)
	if err != nil {
		var oidcErr *authentication.OIDCError
		if errors.As(err, &oidcErr) && oidcErr.RedirectURL != "" {
			ctx.Logger.Warn("OIDC callback error handled with redirect", "message", oidcErr.Message)
			ctx.Redirect(oidcErr.RedirectURL, http.StatusFound)
			return
		}

		ctx.Logger.Error("Failed to handle OIDC callback", "error", err)
		errorURL := fmt.Sprintf("/error?error=%s&error_description=%s", url.QueryEscape("server error"), url.QueryEscape("authentication failed"))
		ctx.Redirect(errorURL, http.StatusFound)
		return
	}

	ctx.Logger.Debug("User successfully authenticated",
		"sub", user.Sub,
		"iss", user.Iss,
		"username", user.Username,
		"email", RedactEmail(user.Email),
	)

	resultUser, err := ctx.Storage.UpsertUser(ctx, user.Sub, user.Iss, user.Username, user.DisplayName, user.Email, user.Groups)
	if err != nil {
		ctx.Logger.Error("Failed to upsert user to database",
			"err", err,
			"iss", user.Iss,
			"sub", user.Sub,
			"username", user.Username,
		)

		errorURL := fmt.Sprintf("/error?error=%s&error_description=%s",
			url.QueryEscape("server_error"),
			url.QueryEscape("Authentication succeeded but failed to create user account. Please try again or contact support."))
		ctx.Redirect(errorURL, http.StatusFound)
		return
	}

	err = ctx.SessionManager.CreateSessionWithTokenExpiry(ctx, idToken, resultUser)
	if err != nil {
		ctx.Logger.Error("Failed to create session",
			"error", err,
			"iss", user.Iss,
			"sub", user.Sub,
			"username", user.Username,
		)

		errorURL := fmt.Sprintf("/error?error=%s&error_description=%s",
			url.QueryEscape("server_error"),
			url.QueryEscape("Authentication succeeded but failed to create user session. Please try again or contact support."))
		ctx.Redirect(errorURL, http.StatusFound)
		return
	}

	ctx.Logger.Debug("User successfully authenticated",
		"user_id", user.Sub,
		"username", user.Username,
		"email", user.Email,
	)

	redirectTo := ctx.SessionManager.GetRedirectAfterLogin(ctx)
	if redirectTo != "" {
		ctx.Redirect(redirectTo, http.StatusFound)
		return
	}

	ctx.Redirect("/", http.StatusFound)
}
