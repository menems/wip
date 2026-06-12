# prometheus-interceptor
> Add a Prometheus metrics interceptor and /metrics endpoint to the ConnectRPC server

**Created**: 2026-03-12 | **Branch**: feat/prometheus-interceptor

## Steps
1. [backend] Create pkg/metrics interceptor
   → NewInterceptor(reg prometheus.Registerer) implements connect.Interceptor; registers rpc_requests_total CounterVec and rpc_duration_seconds HistogramVec, both labeled {procedure, code}; tests verify counters/histograms increment on unary calls.

2. [backend] Add RouteRegistrar handler to pkg/metrics
   → NewHandler(reg prometheus.Gatherer) implements server.RouteRegistrar; mounts GET /metrics via promhttp.HandlerFor; tests verify 200 and non-empty body.

3. [backend] Wire pkg/metrics into cmd/server/main.go
   → Instantiate prometheus.NewRegistry(), metrics.NewInterceptor, and metrics.NewHandler; add interceptor to connect.WithInterceptors chain and handler to registrars slice.
