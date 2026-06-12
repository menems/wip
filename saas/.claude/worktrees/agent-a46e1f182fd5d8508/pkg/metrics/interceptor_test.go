package metrics_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/menems/saas/gen/api/v1/pb"
	"github.com/menems/saas/gen/api/v1/pb/pbconnect"
	"github.com/menems/saas/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// stubLoginHandler is a minimal ConnectRPC server-side handler for AuthService.
type stubLoginHandler struct {
	err error
}

func (s *stubLoginHandler) Login(_ context.Context, _ *connect.Request[pb.LoginRequest]) (*connect.Response[pb.LoginResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&pb.LoginResponse{Token: "tok"}), nil
}

func (s *stubLoginHandler) GetCurrentUser(_ context.Context, _ *connect.Request[pb.GetCurrentUserRequest]) (*connect.Response[pb.GetCurrentUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}

// collectMetricFamilies gathers all metric families from reg.
func collectMetricFamilies(t *testing.T, reg *prometheus.Registry) map[string]*dto.MetricFamily {
	t.Helper()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	m := make(map[string]*dto.MetricFamily, len(mfs))
	for _, mf := range mfs {
		m[mf.GetName()] = mf
	}
	return m
}

func TestInterceptor_WrapUnary_CounterIncrement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		svcErr   error
		wantCode string
	}{
		{
			name:     "success increments ok counter",
			wantCode: "ok",
		},
		{
			name:     "connect error increments correct code counter",
			svcErr:   connect.NewError(connect.CodeNotFound, fmt.Errorf("not found")),
			wantCode: "not_found",
		},
		{
			name:     "unknown error increments unknown counter",
			svcErr:   fmt.Errorf("unexpected error"),
			wantCode: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			interceptor := metrics.NewInterceptor(reg)

			stub := &stubLoginHandler{err: tt.svcErr}
			path, handler := pbconnect.NewAuthServiceHandler(stub, connect.WithInterceptors(interceptor))
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			client := pbconnect.NewAuthServiceClient(srv.Client(), srv.URL)
			_, _ = client.Login(context.Background(), connect.NewRequest(&pb.LoginRequest{
				Email:    "a@b.com",
				Password: "secret",
			}))

			mfs := collectMetricFamilies(t, reg)

			// Verify rpc_requests_total counter.
			counterFamily, ok := mfs["rpc_requests_total"]
			if !ok {
				t.Fatal("rpc_requests_total metric family not found")
			}
			var found bool
			for _, m := range counterFamily.GetMetric() {
				if labelsMatch(m.GetLabel(), pbconnect.AuthServiceLoginProcedure, tt.wantCode) {
					if m.GetCounter().GetValue() != 1 {
						t.Errorf("rpc_requests_total: got %v, want 1", m.GetCounter().GetValue())
					}
					found = true
					break
				}
			}
			if !found {
				t.Errorf("rpc_requests_total: no metric with procedure=%q code=%q", pbconnect.AuthServiceLoginProcedure, tt.wantCode)
			}

			// Verify rpc_duration_seconds histogram.
			durationFamily, ok := mfs["rpc_duration_seconds"]
			if !ok {
				t.Fatal("rpc_duration_seconds metric family not found")
			}
			var durationFound bool
			for _, m := range durationFamily.GetMetric() {
				if labelsMatch(m.GetLabel(), pbconnect.AuthServiceLoginProcedure, tt.wantCode) {
					if m.GetHistogram().GetSampleCount() != 1 {
						t.Errorf("rpc_duration_seconds sample count: got %v, want 1", m.GetHistogram().GetSampleCount())
					}
					durationFound = true
					break
				}
			}
			if !durationFound {
				t.Errorf("rpc_duration_seconds: no metric with procedure=%q code=%q", pbconnect.AuthServiceLoginProcedure, tt.wantCode)
			}
		})
	}
}

// labelsMatch returns true when the label set contains procedure=p and code=c.
func labelsMatch(labels []*dto.LabelPair, procedure, code string) bool {
	var gotProcedure, gotCode string
	for _, lp := range labels {
		switch lp.GetName() {
		case "procedure":
			gotProcedure = lp.GetValue()
		case "code":
			gotCode = lp.GetValue()
		}
	}
	return gotProcedure == procedure && gotCode == code
}
