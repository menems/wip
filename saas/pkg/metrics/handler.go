package metrics

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler serves the Prometheus /metrics endpoint and implements
// pkg/server.RouteRegistrar.
type Handler struct {
	gatherer prometheus.Gatherer
}

// NewHandler creates a Handler that scrapes metrics from the provided gatherer.
func NewHandler(reg prometheus.Gatherer) *Handler {
	return &Handler{gatherer: reg}
}

// Register mounts GET /metrics on mux. The connect.HandlerOption vararg
// satisfies the RouteRegistrar contract but is intentionally unused because
// this is a plain HTTP endpoint, not a ConnectRPC procedure.
func (h *Handler) Register(mux *http.ServeMux, _ ...connect.HandlerOption) {
	mux.Handle("GET /metrics", promhttp.HandlerFor(h.gatherer, promhttp.HandlerOpts{}))
}
