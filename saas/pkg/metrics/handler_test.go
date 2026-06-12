package metrics_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/menems/saas/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func TestHandler_Returns200(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	h := metrics.NewHandler(reg)

	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestHandler_ReturnsNonEmptyBody(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	// Register a simple gauge so the output is non-empty.
	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge",
		Help: "A gauge used by the handler test.",
	})
	reg.MustRegister(g)
	g.Set(42)

	h := metrics.NewHandler(reg)

	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mux.ServeHTTP(w, r)

	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty response body from /metrics")
	}
}
