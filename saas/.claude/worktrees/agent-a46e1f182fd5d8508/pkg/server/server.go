package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// RouteRegistrar is implemented by each feature handler to mount its routes.
type RouteRegistrar interface {
	Register(mux *http.ServeMux, opts ...connect.HandlerOption)
}

// Server wraps an http.Server with h2c support and graceful shutdown.
type Server struct {
	addr              string
	readHeaderTimeout time.Duration
	registrars        []RouteRegistrar
	connectOpts       []connect.HandlerOption
	log               *slog.Logger
}

// Option configures a Server.
type Option func(*Server)

// WithAddr sets the listen address (default ":8080").
func WithAddr(addr string) Option {
	return func(s *Server) {
		s.addr = addr
	}
}

// WithReadHeaderTimeout sets the read-header timeout (default 10s).
func WithReadHeaderTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.readHeaderTimeout = d
	}
}

// WithConnectOptions appends ConnectRPC handler options passed to every RouteRegistrar.Register call.
func WithConnectOptions(opts ...connect.HandlerOption) Option {
	return func(s *Server) {
		s.connectOpts = append(s.connectOpts, opts...)
	}
}

// WithLogger sets the logger used for server lifecycle messages (default: slog.Default()).
func WithLogger(log *slog.Logger) Option {
	return func(s *Server) {
		s.log = log
	}
}

// New creates a Server that will register all provided registrars.
func New(registrars []RouteRegistrar, opts ...Option) *Server {
	s := &Server{
		addr:              ":8080",
		readHeaderTimeout: 10 * time.Second,
		registrars:        registrars,
		log:               slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run starts the HTTP server and blocks until ctx is cancelled, then shuts down
// gracefully with a 5-second timeout. It returns any error from Shutdown.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	for _, r := range s.registrars {
		r.Register(mux, s.connectOpts...)
	}

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: s.readHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	s.log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
