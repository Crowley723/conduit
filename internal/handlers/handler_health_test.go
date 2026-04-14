// internal/handlers/handler_health_test.go
package handlers

import (
	"github.com/Crowley723/conduit/internal/middlewares"
	"github.com/Crowley723/conduit/internal/testutil"
	"testing"
)

func TestHandlerHealth(t *testing.T) {
	tc := testutil.NewTestContextWithURL(t, "GET", "/health")

	tc.CallHandler(HandlerHealth)

	tc.AssertStatus(t, 200)
	tc.AssertContentType(t, "application/json")
	tc.AssertJSONField(t, "status", "OK")
}

func TestHandlerError(t *testing.T) {
	tc := testutil.NewTestContextWithURL(t, "GET", "/error")

	errorHandler := func(ctx *middlewares.AppContext) {
		ctx.SetJSONError(400, "Bad Request")
	}

	tc.CallHandler(errorHandler)

	tc.AssertStatus(t, 400)
	tc.AssertJSONField(t, "error", "Bad Request")
}
