package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/menems/saas/gen/api/v1/pb"
	"github.com/menems/saas/gen/api/v1/pb/pbconnect"
	"github.com/menems/saas/internal/auth"
)

// claimsInjector is a test interceptor that injects claims into the server-side context.
type claimsInjector struct {
	claims *auth.Claims
}

func (ci *claimsInjector) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if ci.claims != nil {
			ctx = auth.ContextWithClaims(ctx, *ci.claims)
		}
		return next(ctx, req)
	}
}

func (ci *claimsInjector) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (ci *claimsInjector) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// mockAuthService implements auth.AuthService for handler tests.
type mockAuthService struct {
	loginFn       func(ctx context.Context, email, password string) (string, error)
	verifyTokenFn func(ctx context.Context, token string) (auth.Claims, error)
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (string, error) {
	return m.loginFn(ctx, email, password)
}

func (m *mockAuthService) VerifyToken(ctx context.Context, token string) (auth.Claims, error) {
	return m.verifyTokenFn(ctx, token)
}

func TestHandler_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		email    string
		password string
		svc      *mockAuthService
		wantCode connect.Code
		wantErr  bool
	}{
		{
			name:     "success",
			email:    "alice@example.com",
			password: "securepassword",
			svc: &mockAuthService{
				loginFn: func(_ context.Context, _, _ string) (string, error) {
					return "jwt-token-123", nil
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid credentials",
			email:    "alice@example.com",
			password: "wrongpassword",
			svc: &mockAuthService{
				loginFn: func(_ context.Context, _, _ string) (string, error) {
					return "", errors.New("login: invalid credentials")
				},
			},
			wantCode: connect.CodeUnauthenticated,
			wantErr:  true,
		},
		{
			name:     "empty email",
			email:    "",
			password: "securepassword",
			svc: &mockAuthService{
				loginFn: func(_ context.Context, _, _ string) (string, error) {
					return "", errors.New("should not be called")
				},
			},
			wantCode: connect.CodeInvalidArgument,
			wantErr:  true,
		},
		{
			name:     "empty password",
			email:    "alice@example.com",
			password: "",
			svc: &mockAuthService{
				loginFn: func(_ context.Context, _, _ string) (string, error) {
					return "", errors.New("should not be called")
				},
			},
			wantCode: connect.CodeInvalidArgument,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := auth.NewHandler(tt.svc)
			path, h := pbconnect.NewAuthServiceHandler(handler)
			mux := http.NewServeMux()
			mux.Handle(path, h)
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			client := pbconnect.NewAuthServiceClient(srv.Client(), srv.URL)
			resp, err := client.Login(context.Background(), connect.NewRequest(&pb.LoginRequest{
				Email:    tt.email,
				Password: tt.password,
			}))

			if tt.wantErr {
				if err == nil {
					t.Fatal("Login() expected error, got nil")
				}
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					if connectErr.Code() != tt.wantCode {
						t.Errorf("Login() code = %v, want %v", connectErr.Code(), tt.wantCode)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("Login() unexpected error: %v", err)
			}
			if resp.Msg.GetToken() != "jwt-token-123" {
				t.Errorf("Login() token = %q, want %q", resp.Msg.GetToken(), "jwt-token-123")
			}
		})
	}
}

func TestHandler_GetCurrentUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		claims   *auth.Claims
		wantCode connect.Code
		wantErr  bool
		wantID   string
		wantMail string
		wantRole string
	}{
		{
			name: "success with claims in context",
			claims: &auth.Claims{
				UserID: "user-456",
				Email:  "bob@example.com",
				Role:   "admin",
			},
			wantErr:  false,
			wantID:   "user-456",
			wantMail: "bob@example.com",
			wantRole: "admin",
		},
		{
			name:     "no claims in context",
			claims:   nil,
			wantCode: connect.CodeUnauthenticated,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &mockAuthService{
				loginFn:       func(_ context.Context, _, _ string) (string, error) { return "", nil },
				verifyTokenFn: func(_ context.Context, _ string) (auth.Claims, error) { return auth.Claims{}, nil },
			}
			injector := &claimsInjector{claims: tt.claims}
			handler := auth.NewHandler(svc)
			path, h := pbconnect.NewAuthServiceHandler(handler, connect.WithInterceptors(injector))
			mux := http.NewServeMux()
			mux.Handle(path, h)
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			client := pbconnect.NewAuthServiceClient(srv.Client(), srv.URL)
			resp, err := client.GetCurrentUser(context.Background(), connect.NewRequest(&pb.GetCurrentUserRequest{}))

			if tt.wantErr {
				if err == nil {
					t.Fatal("GetCurrentUser() expected error, got nil")
				}
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					if connectErr.Code() != tt.wantCode {
						t.Errorf("GetCurrentUser() code = %v, want %v", connectErr.Code(), tt.wantCode)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("GetCurrentUser() unexpected error: %v", err)
			}
			if resp.Msg.GetUserId() != tt.wantID {
				t.Errorf("GetCurrentUser() user_id = %q, want %q", resp.Msg.GetUserId(), tt.wantID)
			}
			if resp.Msg.GetEmail() != tt.wantMail {
				t.Errorf("GetCurrentUser() email = %q, want %q", resp.Msg.GetEmail(), tt.wantMail)
			}
			if resp.Msg.GetRole() != tt.wantRole {
				t.Errorf("GetCurrentUser() role = %q, want %q", resp.Msg.GetRole(), tt.wantRole)
			}
		})
	}
}
