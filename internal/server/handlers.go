package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/Crowley723/conduit/internal/handlers"
	"github.com/Crowley723/conduit/internal/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func setupRouter(ctx *middlewares.AppContext) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middlewares.ClientIPMiddleware)
	//r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(ctx.SessionManager.LoadAndSave)

	r.Use(middlewares.AppContextMiddleware(ctx))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   ctx.Config.CORS.AllowedOrigins,
		AllowedMethods:   ctx.Config.CORS.AllowedMethods,
		AllowedHeaders:   ctx.Config.CORS.AllowedHeaders,
		ExposedHeaders:   ctx.Config.CORS.ExposedHeaders,
		AllowCredentials: ctx.Config.CORS.AllowCredentials,
		MaxAge:           ctx.Config.CORS.MaxAgeSeconds,
	}))

	r.Use(middleware.Compress(5))

	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/dist/assets"))))
	r.Handle("/favicon.ico", http.FileServer(http.Dir("web/dist")))

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/dist/index.html")
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(middlewares.OptionalAuth)
		r.Route("/auth", func(r chi.Router) {
			r.Get("/status", ctx.HandlerFunc(handlers.GETAuthStatusHandler))
			r.Get("/login", ctx.HandlerFunc(handlers.GETLoginHandler))
			r.Get("/callback", ctx.HandlerFunc(handlers.GETCallbackHandler))
			r.Post("/logout", ctx.HandlerFunc(handlers.POSTLogoutHandler))
		})

		if ctx.Config.Storage != nil {
			r.Route("/service-accounts", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(middlewares.RequireCookieAuth)
					r.Get("/", ctx.HandlerFunc(handlers.GETServiceAccounts))
					r.Post("/", ctx.HandlerFunc(handlers.POSTServiceAccount))
					r.Delete("/", ctx.HandlerFunc(handlers.DELETEServiceAccount))
					r.Patch("/pause", ctx.HandlerFunc(handlers.PATCHServiceAccountPause))
					r.Patch("/unpause", ctx.HandlerFunc(handlers.PATCHServiceAccountUnpause))
					r.Get("/scopes", ctx.HandlerFunc(handlers.GETUserScopes))
				})
				r.Group(func(r chi.Router) {
					r.Use(middlewares.RequireServiceAccountAuth)
					r.Get("/whoami", ctx.HandlerFunc(handlers.GETServiceAccountWhoami))
				})
			})
		}

		if ctx.Config.Storage != nil && ctx.Config.MTLS.Enabled {
			r.Route("/certificates", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(middlewares.RequireAuth)
					r.Post("/request", ctx.HandlerFunc(handlers.POSTCertificateRequest))
					r.Get("/my-requests", ctx.HandlerFunc(handlers.GETUserCertificateRequests))
					r.Get("/request/{id}", ctx.HandlerFunc(handlers.GETCertificateRequest))
					r.Get("/{id}/download", ctx.HandlerFunc(handlers.GETCertificateDownload))
					r.Post("/{id}/unlock", ctx.HandlerFunc(handlers.POSTCertificateUnlock))
				})

				r.Group(func(r chi.Router) {
					r.Use(middlewares.RequireAuth)
					r.Get("/requests", ctx.HandlerFunc(handlers.GETCertificateRequests))
					r.Post("/requests/{id}/review", ctx.HandlerFunc(handlers.POSTCertificateReview))
				})
			})
		}

		r.Route("/v1", func(r chi.Router) {
			r.Get("/health", ctx.HandlerFunc(handlers.HandlerHealth))
		})
	})

	return r
}

func setupDebugRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	//r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Mount("/debug", middleware.Profiler())

	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	return r
}
