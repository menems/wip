package httpd

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMiddleware returns an HTTP middleware that records three metrics for
// every request using the provided prometheus.Registerer:
//
//   - http_requests_total          CounterVec  {method, path}
//   - http_request_duration_seconds HistogramVec {method, path} (DefBuckets)
//   - http_requests_in_flight      GaugeVec    {method, path}
func PrometheusMiddleware(reg prometheus.Registerer) func(http.Handler) http.Handler {
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests completed.",
	}, []string{"method", "path"})

	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	inFlight := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of HTTP requests currently being served.",
	}, []string{"method", "path"})

	reg.MustRegister(requests, duration, inFlight)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			labels := prometheus.Labels{"method": r.Method, "path": r.URL.Path}

			inFlight.With(labels).Inc()
			start := time.Now()

			next.ServeHTTP(w, r)

			duration.With(labels).Observe(time.Since(start).Seconds())
			requests.With(labels).Inc()
			inFlight.With(labels).Dec()
		})
	}
}
