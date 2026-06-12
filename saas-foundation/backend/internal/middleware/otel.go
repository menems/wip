package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName    = "github.com/your-org/saas-foundation/backend/internal/middleware"
	instrumentationVersion = "1.0.0"
)

// OTel returns middleware that records an OpenTelemetry trace span and an
// http.server.request.duration histogram measurement for every HTTP request.
//
// Span naming: "METHOD /route/pattern" where the chi route pattern is resolved
// after the downstream handler returns (so {params} are normalised). For
// unmatched routes (404) the raw request path is used as a fallback.
//
// Tracing and metrics use whichever providers are registered globally at the
// time the middleware is first used. In no-op or test environments (where no
// SDK has been initialised) no data is emitted.
func OTel() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Instrument and tracer are obtained once at middleware creation time.
		// Both are safe for concurrent use across requests.
		meter := otel.GetMeterProvider().Meter(
			instrumentationName,
			metric.WithInstrumentationVersion(instrumentationVersion),
		)

		requestDuration, _ := meter.Float64Histogram(
			semconv.HTTPServerRequestDurationName,
			metric.WithDescription(semconv.HTTPServerRequestDurationDescription),
			metric.WithUnit(semconv.HTTPServerRequestDurationUnit),
		)

		tracer := otel.GetTracerProvider().Tracer(
			instrumentationName,
			trace.WithInstrumentationVersion(instrumentationVersion),
		)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract any upstream W3C trace context from the incoming headers.
			propagator := otel.GetTextMapPropagator()
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			start := time.Now()

			// Wrap the ResponseWriter to capture the status code written by
			// the downstream handler.
			rw := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

			// Start a server span with a preliminary name; the chi route pattern
			// is set after next.ServeHTTP() resolves the route.
			ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.ServerAddress(r.Host),
				),
			)
			defer span.End()

			// Propagate the trace context to downstream handlers.
			next.ServeHTTP(rw, r.WithContext(ctx))

			elapsed := time.Since(start).Seconds()
			status := rw.status

			// Read the chi route pattern now that routing has completed.
			// chi stores a *Context pointer in the request context; mutating it
			// during routing is visible here because we share the same pointer.
			route := chiRoutePattern(r)

			// Update span name and attributes with the resolved route.
			if route != "" {
				span.SetName(r.Method + " " + route)
			}
			span.SetAttributes(
				semconv.HTTPResponseStatusCode(status),
				semconv.HTTPRoute(route),
			)

			// Mark 5xx responses as span errors.
			if status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, strconv.Itoa(status))
			}

			// Record the request duration with the key cardinality-bounded attributes.
			requestDuration.Record(ctx, elapsed, metric.WithAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.HTTPResponseStatusCode(status),
				semconv.HTTPRoute(route),
			))
		})
	}
}

// chiRoutePattern returns the matched chi route pattern for the current request,
// e.g. "/api/v1/users/{id}". Returns an empty string when no pattern is available
// (unmatched routes that produce a 404 response).
func chiRoutePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return ""
	}
	return rctx.RoutePattern()
}

// responseRecorder wraps http.ResponseWriter to capture the HTTP status code
// written by the downstream handler.
type responseRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// WriteHeader captures the status code and forwards it to the underlying writer.
func (rw *responseRecorder) WriteHeader(status int) {
	if !rw.wroteHeader {
		rw.status = status
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(status)
}

// Write ensures the status is recorded even when WriteHeader is never called
// explicitly (implicit 200 on the first Write call).
func (rw *responseRecorder) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Unwrap exposes the underlying ResponseWriter so that callers that check for
// optional interfaces (http.Flusher, http.Hijacker, etc.) can still access them.
func (rw *responseRecorder) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// Ensure responseRecorder satisfies the http.ResponseWriter interface.
var _ http.ResponseWriter = (*responseRecorder)(nil)
