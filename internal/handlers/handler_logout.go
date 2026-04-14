package handlers

import (
	"github.com/Crowley723/conduit/internal/middlewares"
	"net/http"
)

func POSTLogoutHandler(ctx *middlewares.AppContext) {
	if !ctx.SessionManager.IsUserAuthenticated(ctx) {
		ctx.SetJSONError(http.StatusBadRequest, "Bad Request")
		return
	}

	user, ok := ctx.SessionManager.GetUser(ctx)
	if !ok || user == nil {
		ctx.Logger.Error("Failed to retrieve user session")
		ctx.SetJSONError(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	err := ctx.SessionManager.Logout(ctx)
	if err != nil {
		ctx.Logger.Error("Failed to logout user", "error", err)
		ctx.SetJSONError(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	ctx.SetJSONStatus(http.StatusOK, "OK")
}
