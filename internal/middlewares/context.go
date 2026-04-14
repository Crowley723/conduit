package middlewares

import (
	"context"
	"encoding/json"
	"github.com/Crowley723/conduit/internal/config"
	"github.com/Crowley723/conduit/internal/services/certificate"
	"github.com/Crowley723/conduit/internal/storage"
	"log/slog"
	"net/http"
)

type AppContext struct {
	context.Context
	Config             *config.Config
	Logger             *slog.Logger
	SessionManager     SessionProvider
	OIDCProvider       OIDCProvider
	Storage            storage.Provider
	CertificateManager certificate.Provider

	principal Principal

	Request  *http.Request
	Response http.ResponseWriter
}

type contextKey string

const appContextKey contextKey = "appContext"

func AppContextMiddleware(baseCtx *AppContext) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCtx := &AppContext{
				Context:            r.Context(),
				Config:             baseCtx.Config,
				Logger:             baseCtx.Logger,
				SessionManager:     baseCtx.SessionManager,
				OIDCProvider:       baseCtx.OIDCProvider,
				Storage:            baseCtx.Storage,
				CertificateManager: baseCtx.CertificateManager,
				principal:          baseCtx.principal,
				Request:            r,
				Response:           w,
			}

			ctx := context.WithValue(r.Context(), appContextKey, requestCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type AppHandler func(*AppContext)

// Handler converts an AppHandler to an http.Handler
func (ctx *AppContext) Handler(h AppHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appCtx := GetAppContext(r)
		if appCtx == nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		h(appCtx)
	})
}

// HandlerFunc converts AppHandler to a http.HandlerFunc
func (ctx *AppContext) HandlerFunc(h AppHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appCtx := GetAppContext(r)
		if appCtx == nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		h(appCtx)
	}
}

func (ctx *AppContext) Redirect(url string, status int) {
	http.Redirect(ctx.Response, ctx.Request, url, status)
}

func NewAppContext(ctx context.Context, cfg *config.Config, logger *slog.Logger, sessionManager SessionProvider, oidcProvider OIDCProvider, storage storage.Provider, certificates certificate.Provider) *AppContext {
	return &AppContext{
		Context:            ctx,
		Config:             cfg,
		Logger:             logger,
		SessionManager:     sessionManager,
		OIDCProvider:       oidcProvider,
		Storage:            storage,
		CertificateManager: certificates,
		principal:          nil,
	}
}

func (ctx *AppContext) SetPrincipal(principal Principal) {
	ctx.principal = principal
}

func (ctx *AppContext) GetPrincipal() Principal {
	return ctx.principal
}

func GetAppContext(r *http.Request) *AppContext {
	if ctx, ok := r.Context().Value(appContextKey).(*AppContext); ok {
		return ctx
	}

	return nil
}

func GetLogger(r *http.Request) *slog.Logger {
	if appCtx := GetAppContext(r); appCtx != nil {
		return appCtx.Logger
	}

	return nil
}

func GetConfig(r *http.Request) *config.Config {
	if appCtx := GetAppContext(r); appCtx != nil {
		return appCtx.Config
	}

	return nil
}

func (ctx *AppContext) WriteJSON(status int, data interface{}) {
	ctx.Response.Header().Set("Content-Type", "application/json")
	ctx.Response.WriteHeader(status)
	if err := json.NewEncoder(ctx.Response).Encode(data); err != nil {
		ctx.Logger.Error("failed to marshal json", "error", err)
	}
}

func (ctx *AppContext) WriteText(status int, text string) {
	ctx.Response.WriteHeader(status)
	if _, err := ctx.Response.Write([]byte(text)); err != nil {
		ctx.Logger.Error("failed to marshal json", "error", err)
	}
}

func (ctx *AppContext) SetJSONError(status int, message string) {
	ctx.WriteJSON(status, map[string]string{
		"error": message,
	})
}

func (ctx *AppContext) SetJSONStatus(status int, message string) {
	ctx.WriteJSON(status, map[string]string{
		"status": message,
	})
}
