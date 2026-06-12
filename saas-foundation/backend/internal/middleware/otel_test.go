package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestProviders creates in-memory TracerProvider and MeterProvider, installs
// the W3C TraceContext propagator, and sets all three as globals for the
// duration of the test. A cleanup function restores the previous globals.
func newTestProviders(t *testing.T) (spanRecorder *tracetest.SpanRecorder, reader *sdkmetric.ManualReader) {
	t.Helper()

	spanRecorder = tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

	reader = sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	prevTP := otel.GetTracerProvider()
	prevMP := otel.GetMeterProvider()
	prevProp := otel.GetTextMapPropagator()

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetMeterProvider(prevMP)
		otel.SetTextMapPropagator(prevProp)
	})

	return spanRecorder, reader
}

// makeChiHandler creates a simple chi router so route patterns are available
// in the request context when the OTel middleware resolves them.
func makeChiHandler(method, pattern string, handler http.HandlerFunc) http.Handler {
	r := chi.NewRouter()
	r.Use(OTel())
	r.Method(method, pattern, handler)
	return r
}

// collectMetrics reads all resource metrics from the ManualReader and returns
// the first histogram data point whose name matches the given instrument name.
func collectHistogramPoints(t *testing.T, reader *sdkmetric.ManualReader, name string) []metricdata.HistogramDataPoint[float64] {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				hist, ok := m.Data.(metricdata.Histogram[float64])
				require.True(t, ok, "expected histogram data type")
				return hist.DataPoints
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestOTel_SpanCreated(t *testing.T) {
	recorder, _ := newTestProviders(t)

	handler := makeChiHandler(http.MethodGet, "/api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	spans := recorder.Ended()
	require.Len(t, spans, 1, "expected exactly one span")

	span := spans[0]
	assert.Equal(t, "GET /api/users/{id}", span.Name(), "span name should use chi route pattern")
	assert.Equal(t, sdktrace.Status{Code: codes.Unset}, span.Status())
}

func TestOTel_SpanHasHTTPAttributes(t *testing.T) {
	recorder, _ := newTestProviders(t)

	handler := makeChiHandler(http.MethodPost, "/api/items", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	span := spans[0]

	// Convert attribute slice to a map for easier lookup.
	attrMap := make(map[string]any)
	for _, kv := range span.Attributes() {
		attrMap[string(kv.Key)] = kv.Value.AsInterface()
	}

	assert.Equal(t, http.MethodPost, attrMap[string(semconv.HTTPRequestMethodKey)], "http.request.method")
	assert.Equal(t, int64(http.StatusCreated), attrMap[string(semconv.HTTPResponseStatusCodeKey)], "http.response.status_code")
	assert.Equal(t, "/api/items", attrMap[string(semconv.HTTPRouteKey)], "http.route")
}

func TestOTel_5xxSetsSpanError(t *testing.T) {
	recorder, _ := newTestProviders(t)

	handler := makeChiHandler(http.MethodGet, "/boom", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status().Code, "5xx should mark span as Error")
}

func TestOTel_4xxDoesNotSetSpanError(t *testing.T) {
	recorder, _ := newTestProviders(t)

	handler := makeChiHandler(http.MethodGet, "/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Unset, spans[0].Status().Code, "4xx should not mark span as Error")
}

func TestOTel_HistogramRecorded(t *testing.T) {
	_, reader := newTestProviders(t)

	handler := makeChiHandler(http.MethodGet, "/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	points := collectHistogramPoints(t, reader, "http.server.request.duration")
	require.Len(t, points, 1, "expected one histogram data point")

	dp := points[0]
	assert.Equal(t, uint64(1), dp.Count, "expected one observation")
	assert.Greater(t, dp.Sum, float64(0), "duration must be positive")
}

func TestOTel_HistogramAttributes(t *testing.T) {
	_, reader := newTestProviders(t)

	handler := makeChiHandler(http.MethodDelete, "/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodDelete, "/items/7", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	points := collectHistogramPoints(t, reader, "http.server.request.duration")
	require.Len(t, points, 1)

	attrMap := make(map[string]any)
	for _, kv := range points[0].Attributes.ToSlice() {
		attrMap[string(kv.Key)] = kv.Value.AsInterface()
	}

	assert.Equal(t, http.MethodDelete, attrMap[string(semconv.HTTPRequestMethodKey)])
	assert.Equal(t, int64(http.StatusNoContent), attrMap[string(semconv.HTTPResponseStatusCodeKey)])
	assert.Equal(t, "/items/{id}", attrMap[string(semconv.HTTPRouteKey)])
}

func TestOTel_W3CTraceContextPropagation(t *testing.T) {
	recorder, _ := newTestProviders(t)

	handler := makeChiHandler(http.MethodGet, "/trace", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Construct a valid W3C traceparent header for a parent span.
	// Format: 00-<traceID>-<spanID>-<flags>
	parentTraceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	parentSpanID := "00f067aa0ba902b7"
	traceparent := "00-" + parentTraceID + "-" + parentSpanID + "-01"

	req := httptest.NewRequest(http.MethodGet, "/trace", nil)
	req.Header.Set("traceparent", traceparent)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := recorder.Ended()
	require.Len(t, spans, 1)

	spanCtx := spans[0].Parent()
	assert.True(t, spanCtx.IsValid(), "span should have a valid parent context from W3C header")
	assert.Equal(t, parentTraceID, spanCtx.TraceID().String())
	assert.Equal(t, parentSpanID, spanCtx.SpanID().String())
}

func TestOTel_ImplicitStatus200(t *testing.T) {
	recorder, _ := newTestProviders(t)

	// Handler writes body without calling WriteHeader explicitly → implicit 200.
	handler := makeChiHandler(http.MethodGet, "/implicit", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	})

	req := httptest.NewRequest(http.MethodGet, "/implicit", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := recorder.Ended()
	require.Len(t, spans, 1)

	attrMap := make(map[string]any)
	for _, kv := range spans[0].Attributes() {
		attrMap[string(kv.Key)] = kv.Value.AsInterface()
	}
	assert.Equal(t, int64(http.StatusOK), attrMap[string(semconv.HTTPResponseStatusCodeKey)])
}

// ---------------------------------------------------------------------------
// responseRecorder unit tests
// ---------------------------------------------------------------------------

func TestResponseRecorder_CapturesStatus(t *testing.T) {
	rw := httptest.NewRecorder()
	rec := &responseRecorder{ResponseWriter: rw, status: http.StatusOK}

	rec.WriteHeader(http.StatusTeapot)
	assert.Equal(t, http.StatusTeapot, rec.status)
	assert.Equal(t, http.StatusTeapot, rw.Code)
}

func TestResponseRecorder_WriteImplies200(t *testing.T) {
	rw := httptest.NewRecorder()
	rec := &responseRecorder{ResponseWriter: rw, status: http.StatusOK}

	_, _ = rec.Write([]byte("body"))
	assert.Equal(t, http.StatusOK, rec.status)
}

func TestResponseRecorder_WriteHeaderIdempotent(t *testing.T) {
	rw := httptest.NewRecorder()
	rec := &responseRecorder{ResponseWriter: rw, status: http.StatusOK}

	rec.WriteHeader(http.StatusAccepted)
	rec.WriteHeader(http.StatusBadRequest) // should be ignored
	assert.Equal(t, http.StatusAccepted, rec.status, "first WriteHeader wins")
}
