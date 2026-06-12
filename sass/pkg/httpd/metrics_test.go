package httpd_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/menems/sass/pkg/httpd"
)

func TestMetricsEndpoint_contentType(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	srv := httpd.New(":0", httpd.WithLogger(discardLogger), httpd.WithPrometheusRegistry(reg))

	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)

	resp := w.Result()
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want it to contain %q", ct, "text/plain")
	}
	if !strings.Contains(ct, "version=0.0.4") {
		t.Errorf("Content-Type = %q, want it to contain %q", ct, "version=0.0.4")
	}
}

func TestMetricsEndpoint_countsRequest(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	srv := httpd.New(":0", httpd.WithLogger(discardLogger), httpd.WithPrometheusRegistry(reg))

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	// Fire one request to /health/live to produce metric observations.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+"/health/live", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	// Now fetch /metrics and verify the metric names appear in the output.
	metricsReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+"/metrics", nil)
	if err != nil {
		t.Fatalf("new metrics request: %v", err)
	}
	metricsResp, err := http.DefaultClient.Do(metricsReq)
	if err != nil {
		t.Fatalf("do metrics request: %v", err)
	}
	t.Cleanup(func() { _ = metricsResp.Body.Close() })

	body, err := io.ReadAll(metricsResp.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	bodyStr := string(body)

	for _, want := range []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"http_requests_in_flight",
	} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("metrics body does not contain %q", want)
		}
	}
}

func TestPrometheusMiddleware_noRegistryConflict(t *testing.T) {
	t.Parallel()

	// Two servers with independent registries must not panic on MustRegister.
	reg1 := prometheus.NewRegistry()
	reg2 := prometheus.NewRegistry()

	srv1 := httpd.New(":0", httpd.WithLogger(discardLogger), httpd.WithPrometheusRegistry(reg1))
	srv2 := httpd.New(":0", httpd.WithLogger(discardLogger), httpd.WithPrometheusRegistry(reg2))

	for _, srv := range []*httpd.Server{srv1, srv2} {
		r := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	}
}
