package handlers

import (
	"github.com/Crowley723/conduit/internal/middlewares"
	"net/http"
	"net/url"
	"strings"
)

func GETLoginHandler(ctx *middlewares.AppContext) {
	if ctx.SessionManager.IsAuthenticated(ctx) {
		ctx.Logger.Debug("User already authenticated")
		ctx.SetJSONStatus(http.StatusOK, "ok")
		return
	}

	redirectTo := ctx.Request.URL.Query().Get("rd")
	if redirectTo == "" {
		redirectTo = ctx.Request.Header.Get("Referer")
		if redirectTo == "" {
			redirectTo = "/"
		}
	}

	if redirectTo != "/" {
		parsedUrl, err := url.Parse(redirectTo)
		if err != nil || parsedUrl.IsAbs() || parsedUrl.Host != "" {
			//ctx.Logger.Warn("Invalid redirect URL, using root", "redirect", redirectTo)
			redirectTo = "/"
		}

		if !strings.HasPrefix(redirectTo, "/") {
			redirectTo = "/" + redirectTo
		}
	}

	if strings.Contains(redirectTo, "/error") {
		ctx.Logger.Debug("Referer is error page, redirecting to root instead", "original_referer", redirectTo)
		redirectTo = "/"
	}

	ctx.SessionManager.SetRedirectAfterLogin(ctx, redirectTo)

	authURL, err := ctx.OIDCProvider.StartLogin(ctx)
	if err != nil {
		ctx.Logger.Error("Failed to start login", "error", err)
		ctx.SetJSONError(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	ctx.Logger.Debug("Redirecting to OIDC Provider", "url", authURL)

	ctx.WriteJSON(http.StatusOK, map[string]string{
		"status":       "redirect_required",
		"redirect_url": authURL,
	})
}
