// Package httpd provides a production-ready HTTP server with graceful shutdown.
package httpd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/menems/sass/pkg/telemetry"
)

// Server is an HTTP server. It is safe for concurrent use.
type Server struct {
	addr            string
	readTimeout     time.Duration
	writeTimeout    time.Duration
	idleTimeout     time.Duration
	shutdownTimeout time.Duration
	logger          *slog.Logger
	promRegistry    *prometheus.Registry
	promMiddleware  func(http.Handler) http.Handler
	otelMiddleware  func(http.Handler) http.Handler
	checkers        []Checker
	routeFn         func(chi.Router)
}

// Option configures a Server.
type Option func(*Server)

// WithLogger sets the structured logger. Defaults to slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) { s.logger = l }
}

// WithReadTimeout sets the maximum duration for reading the full request.
// Defaults to 5s.
func WithReadTimeout(d time.Duration) Option {
	return func(s *Server) { s.readTimeout = d }
}

// WithWriteTimeout sets the maximum duration before timing out writes of the response.
// Defaults to 10s.
func WithWriteTimeout(d time.Duration) Option {
	return func(s *Server) { s.writeTimeout = d }
}

// WithIdleTimeout sets the maximum time to wait for the next request on a keep-alive connection.
// Defaults to 120s.
func WithIdleTimeout(d time.Duration) Option {
	return func(s *Server) { s.idleTimeout = d }
}

// WithShutdownTimeout sets the maximum time allowed for in-flight requests to complete
// during a graceful shutdown. Defaults to 10s.
func WithShutdownTimeout(d time.Duration) Option {
	return func(s *Server) { s.shutdownTimeout = d }
}

// WithTracerProvider sets an OpenTelemetry TracerProvider for the server.
// Each request will produce one OTEL span with the request_id attribute.
// When not set, tracing is a no-op.
func WithTracerProvider(tp *sdktrace.TracerProvider) Option {
	return func(s *Server) {
		s.otelMiddleware = telemetry.OtelMiddleware(tp)
	}
}

// WithCheckers registers one or more Checkers that are run on GET /health/ready.
// All checkers are executed concurrently; any error causes a 503 response.
func WithCheckers(checkers ...Checker) Option {
	return func(s *Server) { s.checkers = append(s.checkers, checkers...) }
}

// WithRoutes registers an application route-mounting function called inside Handler().
func WithRoutes(fn func(chi.Router)) Option {
	return func(s *Server) { s.routeFn = fn }
}

// WithPrometheusRegistry sets a custom prometheus.Registry for the server.
// Use this to isolate metric registration across multiple servers (e.g. in tests).
// Defaults to a fresh prometheus.NewRegistry().
func WithPrometheusRegistry(reg *prometheus.Registry) Option {
	return func(s *Server) {
		s.promRegistry = reg
		s.promMiddleware = PrometheusMiddleware(reg)
	}
}

// New creates a new Server listening on addr, configured by opts.
func New(addr string, opts ...Option) *Server {
	reg := prometheus.NewRegistry()
	s := &Server{
		addr:            addr,
		readTimeout:     5 * time.Second,
		writeTimeout:    10 * time.Second,
		idleTimeout:     120 * time.Second,
		shutdownTimeout: 10 * time.Second,
		logger:          slog.Default(),
		promRegistry:    reg,
		promMiddleware:  PrometheusMiddleware(reg),
		otelMiddleware:  func(next http.Handler) http.Handler { return next }, // no-op
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Handler returns the chi router for the server.
// Useful for embedding in a larger mux or for testing.
func (s *Server) Handler() chi.Router {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID, SlogMiddleware(s.logger), s.promMiddleware, s.otelMiddleware)
	r.Get("/metrics", promhttp.HandlerFor(s.promRegistry, promhttp.HandlerOpts{}).ServeHTTP)
	r.Get("/health/live", s.handleLive())
	r.Get("/health/ready", s.handleReady(s.checkers))
	r.Get("/", s.handleRoot())
	if s.routeFn != nil {
		s.routeFn(r)
	}
	return r
}

// Run starts the HTTP server and blocks until ctx is cancelled or a listen error occurs.
// On cancellation it performs a graceful shutdown, waiting up to the configured
// shutdown timeout for in-flight requests to complete.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:         s.addr,
		Handler:      s.Handler(),
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
		IdleTimeout:  s.idleTimeout,
	}

	shutErr := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		shutErr <- srv.Shutdown(shutCtx)
	}()

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen: %w", err)
	}
	if err := <-shutErr; err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

func (s *Server) handleRoot() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, world!\n"))
	}
}
