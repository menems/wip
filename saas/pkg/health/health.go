package health

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
)

// Checker is implemented by any dependency that can report its readiness.
type Checker interface {
	// Ping returns a non-nil error when the dependency is unavailable.
	Ping(ctx context.Context) error
}

// Handler serves Kubernetes liveness and readiness probe endpoints and
// implements pkg/server.RouteRegistrar.
type Handler struct {
	checkers []Checker
}

// New creates a Handler. Zero or more Checkers may be provided; all must
// return nil from Ping for the readiness probe to report healthy.
func New(checkers ...Checker) *Handler {
	return &Handler{checkers: checkers}
}

// Register mounts the probe routes on mux. The connect.HandlerOption vararg
// satisfies the RouteRegistrar contract but is intentionally unused because
// these are plain HTTP endpoints, not ConnectRPC procedures.
func (h *Handler) Register(mux *http.ServeMux, _ ...connect.HandlerOption) {
	mux.HandleFunc("GET /healthz/live", h.live)
	mux.HandleFunc("GET /healthz/ready", h.ready)
}

// live always returns 200 OK — the pod is alive as long as it can handle HTTP.
func (h *Handler) live(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// ready returns 200 OK when every Checker is healthy, or 503 Service
// Unavailable on the first failure.
func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	for _, c := range h.checkers {
		if err := c.Ping(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
