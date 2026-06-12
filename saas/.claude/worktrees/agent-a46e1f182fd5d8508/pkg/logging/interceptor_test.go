package logging_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/menems/saas/gen/api/v1/pb"
	"github.com/menems/saas/gen/api/v1/pb/pbconnect"
	"github.com/menems/saas/pkg/logging"
)

// fakeHandler is a slog.Handler that captures every log record.
type fakeHandler struct {
	records []slog.Record
}

func (h *fakeHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *fakeHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *fakeHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *fakeHandler) WithGroup(_ string) slog.Handler      { return h }

// attrMap returns the top-level string attributes of a slog.Record as a map.
func attrMap(r slog.Record) map[string]string {
	m := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.String()
		return true
	})
	return m
}

// stubLoginHandler is a minimal ConnectRPC server-side handler for AuthService.
type stubLoginHandler struct {
	err           error
	captureLogger **slog.Logger // if non-nil, the handler writes FromContext result here
}

func (s *stubLoginHandler) Login(ctx context.Context, req *connect.Request[pb.LoginRequest]) (*connect.Response[pb.LoginResponse], error) {
	if s.captureLogger != nil {
		*s.captureLogger = logging.FromContext(ctx)
	}
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&pb.LoginResponse{Token: "tok"}), nil
}

func (s *stubLoginHandler) GetCurrentUser(_ context.Context, _ *connect.Request[pb.GetCurrentUserRequest]) (*connect.Response[pb.GetCurrentUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}

func TestInterceptor_WrapUnary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		svcErr   error
		wantCode string
	}{
		{
			name:     "success logs CodeOK",
			wantCode: "ok",
		},
		{
			name:     "connect error logs correct code",
			svcErr:   connect.NewError(connect.CodeNotFound, fmt.Errorf("not found")),
			wantCode: "not_found",
		},
		{
			name:     "unknown error logs CodeUnknown",
			svcErr:   fmt.Errorf("some unexpected error"),
			wantCode: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := &fakeHandler{}
			logger := slog.New(h)
			interceptor := logging.NewInterceptor(logger)

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

			if len(h.records) != 1 {
				t.Fatalf("expected 1 log record, got %d", len(h.records))
			}

			attrs := attrMap(h.records[0])

			if attrs["procedure"] != pbconnect.AuthServiceLoginProcedure {
				t.Errorf("procedure: got %q, want %q", attrs["procedure"], pbconnect.AuthServiceLoginProcedure)
			}
			if _, ok := attrs["duration"]; !ok {
				t.Error("expected duration attribute in log record")
			}
			if attrs["code"] != tt.wantCode {
				t.Errorf("code: got %q, want %q", attrs["code"], tt.wantCode)
			}
		})
	}
}

func TestInterceptor_InjectsLoggerIntoContext(t *testing.T) {
	t.Parallel()

	h := &fakeHandler{}
	logger := slog.New(h)
	interceptor := logging.NewInterceptor(logger)

	var captured *slog.Logger
	stub := &stubLoginHandler{captureLogger: &captured}
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

	if captured == nil {
		t.Fatal("handler did not capture a logger from context")
	}
	if captured == slog.Default() {
		t.Error("FromContext returned slog.Default(); expected the injected logger")
	}
	if captured != logger {
		t.Errorf("FromContext: got %p, want injected logger %p", captured, logger)
	}
}
