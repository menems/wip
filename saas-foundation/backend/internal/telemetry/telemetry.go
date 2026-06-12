// Package telemetry bootstraps the OpenTelemetry SDK for the application.
//
// Call Setup once at startup. When the OTLP endpoint is empty the function
// returns immediately, leaving the global no-op providers in place so the
// rest of the codebase compiles and runs without any tracing infrastructure.
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Setup initialises the global TracerProvider and MeterProvider and installs
// the W3C TraceContext + Baggage composite propagator.
//
// When endpoint is empty the function is a no-op: the global providers remain
// as the SDK no-ops and the returned shutdown func is a safe no-op too.
//
// The caller must defer the returned shutdown function to flush and close
// exporters cleanly on process exit:
//
//	shutdown, err := telemetry.Setup(ctx, "my-service", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
//	if err != nil { log.Fatal(err) }
//	defer shutdown(ctx)
func Setup(ctx context.Context, serviceName, endpoint string) (shutdown func(context.Context) error, err error) {
	if endpoint == "" {
		// No-op path: nothing to configure, nothing to clean up.
		return func(context.Context) error { return nil }, nil
	}

	var shutdownFns []func(context.Context) error

	// rollback calls all registered shutdown functions in reverse order,
	// collecting any errors that arise.
	rollback := func(ctx context.Context) error {
		var rollbackErr error
		for i := len(shutdownFns) - 1; i >= 0; i-- {
			if e := shutdownFns[i](ctx); e != nil && rollbackErr == nil {
				rollbackErr = e
			}
		}
		return rollbackErr
	}

	// On any setup error call rollback to close any partially-created
	// exporters before returning the error to the caller.
	defer func() {
		if err != nil {
			_ = rollback(ctx)
		}
	}()

	// -------------------------------------------------------------------------
	// Propagator — W3C TraceContext + Baggage
	// -------------------------------------------------------------------------
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// -------------------------------------------------------------------------
	// Resource — identifies the service to the backend
	// -------------------------------------------------------------------------
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: build resource: %w", err)
	}

	// -------------------------------------------------------------------------
	// Trace exporter + provider
	// -------------------------------------------------------------------------
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
	)
	shutdownFns = append(shutdownFns, tp.Shutdown)
	otel.SetTracerProvider(tp)

	// -------------------------------------------------------------------------
	// Metric exporter + provider
	// -------------------------------------------------------------------------
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpointURL(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: create metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(metricExporter,
				sdkmetric.WithInterval(30*time.Second),
			),
		),
		sdkmetric.WithResource(res),
	)
	shutdownFns = append(shutdownFns, mp.Shutdown)
	otel.SetMeterProvider(mp)

	return rollback, nil
}
