package handlers

import (
	"github.com/Crowley723/conduit/internal/middlewares"
)

func HandlerHealth(ctx *middlewares.AppContext) {
	ctx.SetJSONStatus(200, "OK")
}
