package httpd

import (
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// SlogMiddleware returns an HTTP middleware that logs each request as a single
// structured INFO line using the provided logger after the response is written.
// Log fields: method, path, status, duration_ms, request_id.
// It also sets the X-Request-Id response header from the request context,
// which must be populated upstream by chimiddleware.RequestID.
func SlogMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			reqID := chimiddleware.GetReqID(r.Context())
			ww.Header().Set("X-Request-Id", reqID)

			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}

			logger.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.String("request_id", reqID),
			)
		})
	}
}
