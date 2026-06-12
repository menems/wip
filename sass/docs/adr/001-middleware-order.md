# ADR-001: HTTP Middleware Chain Order

**Status:** Accepted
**Date:** 2026-02-27
**Deciders:** menems

---

## Context

`pkg/httpd.Server` uses [go-chi/chi](https://github.com/go-chi/chi) and registers
a stack of cross-cutting middleware in `Handler()`:

```go
r.Use(
    chimiddleware.RequestID,  // 1
    SlogMiddleware(s.logger), // 2
    s.promMiddleware,         // 3
    s.otelMiddleware,         // 4
)
```

Chi middleware executes as an **onion**: the first registered middleware is the
outermost layer. For an incoming request the execution order is therefore:

```
RequestID → Slog → Prometheus → OtelMiddleware → Handler
                                                ↕  (handler runs)
RequestID ← Slog ← Prometheus ← OtelMiddleware ←
```

Each layer depends on state produced by the layers above it. The order is
non-obvious and incorrect reordering silently breaks observability or
correlation across all three pillars (logs, metrics, traces).

---

## Decision

The middleware chain is fixed as:

### Layer 1 — `chimiddleware.RequestID` (outermost)

Generates a unique `X-Request-Id` per request (UUID v4) and stores it in the
request context. Every subsequent middleware reads it via
`chimiddleware.GetReqID(r.Context())`.

**Must be first** because all downstream layers depend on a populated request ID:
- `SlogMiddleware` emits it as the `request_id` log field.
- `OtelMiddleware` attaches it as a span attribute for trace–log correlation.
- `problem.WriteError` returns it in RFC 7807 error bodies.

Placing it anywhere else would leave those fields empty for middleware running
above it.

### Layer 2 — `SlogMiddleware` (structured logger)

Wraps the `http.ResponseWriter` to intercept the final status code, then emits
one structured JSON log line **after** the inner handler returns, with fields:
`method`, `path`, `status`, `duration_ms`, `request_id`.

**Must be second** (inside RequestID, outside everything else) so that:
1. The `duration_ms` field covers the complete handler execution time including
   Prometheus and OTEL overhead.
2. The `status` field reflects the actual response written by the handler, not
   an intermediate state.
3. Logging always fires even if an inner middleware panics; a future Recoverer
   should be inserted between Slog and Prometheus so the panic is still logged.

### Layer 3 — `PrometheusMiddleware` (metrics)

Records three metrics with `{method, path}` labels:

| Metric | Type | When |
|--------|------|------|
| `http_requests_in_flight` | GaugeVec | incremented before / decremented after handler |
| `http_request_duration_seconds` | HistogramVec | observed after handler returns |
| `http_requests_total` | CounterVec | incremented after handler returns |

**Must be inside Slog** so the logger can measure total latency including
metric-recording overhead (negligible but consistent).

**Must be outside OTEL** so metrics are recorded even when tracing is disabled
or the OTLP exporter is unreachable. Prometheus availability must not depend on
the tracing subsystem.

### Layer 4 — `OtelMiddleware` (distributed tracing, innermost)

Extracts an incoming W3C TraceContext from request headers, starts a server-side
span named `"$METHOD $path"`, attaches the `request_id` attribute, and ends the
span after the handler returns. Defaults to a no-op when no `TracerProvider` is
configured.

**Must be innermost** (closest to the handler) so that:
1. The span covers only handler execution, not logging or metrics overhead.
2. Handlers that make outbound calls inherit the active span from the context,
   enabling child spans and full distributed traces.
3. The `request_id` attribute is guaranteed to be set (RequestID already ran).

---

## Consequences

### Positive
- **Correlation is guaranteed**: every log line, every metric sample, and every
  trace span for the same request share the same `request_id`.
- **Metrics are independent of tracing**: Prometheus operates even when the OTLP
  exporter is misconfigured or unreachable.
- **Full-lifecycle timing**: the Slog duration and Prometheus histogram both
  measure the complete inner call stack consistently.

### Negative / Trade-offs
- The order is **implicit** — chi does not enforce or name the layers. Any
  future `r.Use(...)` call that inserts middleware in the wrong position will
  silently break correlation without a compile-time error.
- **No panic recovery middleware** is currently registered. A future `Recoverer`
  must be inserted between `SlogMiddleware` (layer 2) and `PrometheusMiddleware`
  (layer 3) so that panics are logged with a `request_id` before the goroutine
  is reclaimed, and Prometheus records the 500 rather than an incomplete sample.

### Neutral
- Changing the order requires updating this ADR and the tests in
  `pkg/httpd/middleware_test.go` and `pkg/httpd/server_test.go`.
