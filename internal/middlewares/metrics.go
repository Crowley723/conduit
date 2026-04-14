package middlewares

import (
	"github.com/Crowley723/conduit/internal/metrics"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()

		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = r.URL.Path
		}

		metrics.HTTPRequestsTotal.WithLabelValues(
			r.Method,
			routePattern,
			strconv.Itoa(ww.statusCode),
		).Inc()

		metrics.HTTPRequestDuration.WithLabelValues(
			r.Method,
			routePattern,
		).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
