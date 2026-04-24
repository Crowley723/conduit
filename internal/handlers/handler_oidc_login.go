package handlers

import (
	"net/http"

	"github.com/Crowley723/conduit/internal/middlewares"
)

func GETLoginHandler(ctx *middlewares.AppContext) {
	if ctx.SessionManager.IsUserAuthenticated(ctx) {
		ctx.Redirect("/", http.StatusFound)
		return
	}

	authURL, err := ctx.OIDCProvider.StartLogin(ctx)
	if err != nil {
		ctx.Logger.Error("Failed to start login", "error", err)
		ctx.SetJSONError(http.StatusInternalServerError, "Internal Server Error")
		return
	}
	
	ctx.Redirect(authURL, http.StatusFound)
}
