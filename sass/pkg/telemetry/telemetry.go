// Package telemetry provides OpenTelemetry setup for distributed tracing.
package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"os"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// NewTracerProvider creates and configures an OpenTelemetry TracerProvider.
// When OTEL_EXPORTER_OTLP_ENDPOINT is set the provider uses an OTLP HTTP
// exporter; otherwise it falls back to a human-readable stdout exporter for
// local development. The W3C TraceContext propagator is registered globally
// so incoming trace headers are honoured automatically.
func NewTracerProvider(ctx context.Context, serviceName string) (*sdktrace.TracerProvider, error) {
	exp, err := newExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("create exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(newResource(serviceName)),
	)

	// Register W3C TraceContext + Baggage propagators globally.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetTracerProvider(tp)

	return tp, nil
}

// OtelMiddleware returns an HTTP middleware that starts an OpenTelemetry span
// for every request. The span name is "$METHOD $path". The request_id from
// chi's RequestID middleware is added as a span attribute for log correlation.
// Incoming W3C TraceContext headers are extracted to preserve distributed traces.
func OtelMiddleware(tp *sdktrace.TracerProvider) func(http.Handler) http.Handler {
	tracer := tp.Tracer("sass")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract W3C TraceContext from incoming headers.
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			spanName := r.Method + " " + r.URL.Path
			ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
			defer span.End()

			// Propagate request_id for log correlation.
			if reqID := chimiddleware.GetReqID(ctx); reqID != "" {
				span.SetAttributes(attribute.String("request_id", reqID))
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// newResource builds an OTel resource with the service name attribute.
func newResource(serviceName string) *sdkresource.Resource {
	r, _ := sdkresource.Merge(
		sdkresource.Default(),
		sdkresource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	return r
}

// newExporter selects the exporter based on environment.
func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		exp, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("otlp http exporter: %w", err)
		}
		return exp, nil
	}

	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("stdout exporter: %w", err)
	}
	return exp, nil
}
