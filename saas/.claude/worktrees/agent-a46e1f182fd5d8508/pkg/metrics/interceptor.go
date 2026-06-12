package metrics

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
)

// Interceptor is a ConnectRPC interceptor that records Prometheus metrics for
// every RPC call: a request counter and a latency histogram, both labeled by
// {procedure, code}.
type Interceptor struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewInterceptor creates a new metrics interceptor and registers its
// collectors with reg.  It panics (via prometheus.MustRegister) if a
// conflicting collector is already registered.
func NewInterceptor(reg prometheus.Registerer) *Interceptor {
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "rpc_requests_total",
		Help: "Total number of RPC requests, labeled by procedure and status code.",
	}, []string{"procedure", "code"})

	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "rpc_duration_seconds",
		Help:    "Duration of RPC calls in seconds, labeled by procedure and status code.",
		Buckets: prometheus.DefBuckets,
	}, []string{"procedure", "code"})

	reg.MustRegister(requests, duration)

	return &Interceptor{requests: requests, duration: duration}
}

// WrapUnary implements connect.Interceptor.
func (i *Interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		procedure := req.Spec().Procedure

		resp, err := next(ctx, req)

		code := codeString(codeOf(err))
		elapsed := time.Since(start).Seconds()
		i.requests.WithLabelValues(procedure, code).Inc()
		i.duration.WithLabelValues(procedure, code).Observe(elapsed)

		return resp, err
	}
}

// WrapStreamingClient implements connect.Interceptor (no-op for client streams).
func (i *Interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor.
func (i *Interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		procedure := conn.Spec().Procedure

		err := next(ctx, conn)

		code := codeString(codeOf(err))
		elapsed := time.Since(start).Seconds()
		i.requests.WithLabelValues(procedure, code).Inc()
		i.duration.WithLabelValues(procedure, code).Observe(elapsed)

		return err
	}
}
