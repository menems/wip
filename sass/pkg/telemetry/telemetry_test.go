package telemetry_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/menems/sass/pkg/telemetry"
)

func TestNewTracerProvider_noEnv(t *testing.T) {
	// t.Setenv is incompatible with t.Parallel (env is process-global).
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	tp, err := telemetry.NewTracerProvider(context.Background(), "test-svc")
	if err != nil {
		t.Fatalf("NewTracerProvider() error = %v", err)
	}
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	if tp == nil {
		t.Fatal("expected non-nil TracerProvider")
	}
}

func TestNewTracerProvider_otlpEnv(t *testing.T) {
	// t.Setenv is incompatible with t.Parallel (env is process-global).
	// Point at a non-existent endpoint; provider creation must still succeed
	// (connection is lazy — errors surface on export, not on init).
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")

	tp, err := telemetry.NewTracerProvider(context.Background(), "test-svc")
	if err != nil {
		t.Fatalf("NewTracerProvider() with OTLP env error = %v", err)
	}
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	if tp == nil {
		t.Fatal("expected non-nil TracerProvider")
	}
}

func TestOtelMiddleware_spanCreated(t *testing.T) {
	t.Parallel()

	// In-memory exporter captures spans synchronously.
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	// Build a handler stack: RequestID → OtelMiddleware → stub handler.
	stub := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := chimiddleware.RequestID(telemetry.OtelMiddleware(tp)(stub))

	r := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	span := spans[0]

	// Span name must encode method and path.
	wantName := "GET /health/live"
	if span.Name != wantName {
		t.Errorf("span.Name = %q, want %q", span.Name, wantName)
	}

	// request_id attribute must be present and non-empty.
	var reqIDAttr string
	for _, a := range span.Attributes {
		if string(a.Key) == "request_id" {
			reqIDAttr = a.Value.AsString()
		}
	}
	if reqIDAttr == "" {
		t.Error("span missing non-empty request_id attribute")
	}
}
