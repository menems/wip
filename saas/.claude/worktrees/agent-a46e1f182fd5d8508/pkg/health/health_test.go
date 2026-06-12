package health_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/menems/saas/pkg/health"
)

// okChecker is a Checker that always reports healthy.
type okChecker struct{}

func (okChecker) Ping(_ context.Context) error { return nil }

// failChecker is a Checker that always reports unhealthy.
type failChecker struct{ err error }

func (f failChecker) Ping(_ context.Context) error { return f.err }

func TestLiveness(t *testing.T) {
	t.Parallel()

	h := health.New()
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestReadiness_NoCheckers(t *testing.T) {
	t.Parallel()

	h := health.New()
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestReadiness_AllHealthy(t *testing.T) {
	t.Parallel()

	h := health.New(okChecker{}, okChecker{})
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestReadiness_OneUnhealthy(t *testing.T) {
	t.Parallel()

	boom := errors.New("db unavailable")
	h := health.New(okChecker{}, failChecker{err: boom})
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", w.Code)
	}
}

func TestReadiness_FirstCheckerFails(t *testing.T) {
	t.Parallel()

	boom := errors.New("first checker failed")
	h := health.New(failChecker{err: boom}, okChecker{})
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", w.Code)
	}
}
