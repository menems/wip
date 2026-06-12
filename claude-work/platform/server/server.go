package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

)

// Config holds all server configuration parameters.
type Config struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Host:            "127.0.0.1",
		Port:            8080,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}
}

// Server is an HTTP API server.
type Server struct {
	cfg        Config
	mux        *http.ServeMux
	srv        *http.Server
	log        *slog.Logger
	start      time.Time
	ready      chan struct{}
	addr       string
	registrars []RouteRegistrar
}

// New creates a new Server with the given Config, logger, and options.
func New(cfg Config, log *slog.Logger, opts ...Option) *Server {
	s := &Server{
		cfg:   cfg,
		mux:   http.NewServeMux(),
		log:   log,
		ready: make(chan struct{}),
	}
	for _, o := range opts {
		o(s)
	}
	s.routes()
	s.srv = &http.Server{
		Handler:      s.mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	return s
}

// routes registers built-in routes and any routes from registered registrars.
func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.healthHandler())
	s.mux.HandleFunc("POST /echo", s.echoHandler())
	for _, r := range s.registrars {
		r.RegisterRoutes(s.mux)
	}
	s.mux.Handle("/", s.notFoundHandler())
}

// ServeHTTP implements http.Handler, delegating to the mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Addr blocks until the server has successfully started listening and
// returns the address (host:port) it is listening on.
func (s *Server) Addr() string {
	<-s.ready
	return s.addr
}

// Run starts the server and blocks until ctx is cancelled or a fatal error occurs.
func (s *Server) Run(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	s.start = time.Now()
	s.addr = ln.Addr().String()
	s.log.InfoContext(ctx, "listening", "addr", "http://"+s.addr)
	close(s.ready)

	errCh := make(chan error, 1)
	go func() {
		if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	if err := s.srv.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	<-errCh
	s.log.Info("shutdown complete")
	return nil
}
