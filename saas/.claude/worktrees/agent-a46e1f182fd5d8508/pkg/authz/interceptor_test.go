package authz_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/menems/saas/gen/api/v1/pb"
	"github.com/menems/saas/gen/api/v1/pb/pbconnect"
	"github.com/menems/saas/internal/auth"
	"github.com/menems/saas/pkg/authz"
)

// stubVerifier implements authz.TokenVerifier for testing.
type stubVerifier struct {
	claims authz.Claims
	err    error
}

func (s *stubVerifier) VerifyToken(_ context.Context, _ string) (authz.Claims, error) {
	return s.claims, s.err
}

// stubAuthService implements auth.AuthService for testing — only GetCurrentUser matters here.
type stubAuthService struct {
	loginToken string
	loginErr   error
}

func (s *stubAuthService) Login(_ context.Context, _, _ string) (string, error) {
	return s.loginToken, s.loginErr
}

func (s *stubAuthService) VerifyToken(_ context.Context, _ string) (authz.Claims, error) {
	return authz.Claims{}, nil
}

func TestAuthInterceptor(t *testing.T) {
	t.Parallel()

	validClaims := authz.Claims{
		UserID:    "user-1",
		Email:     "test@example.com",
		Role:      "admin",
		ExpiresAt: time.Now().Add(time.Hour),
		IssuedAt:  time.Now(),
	}

	tests := []struct {
		name     string
		token    string
		verifier *stubVerifier
		public   bool
		wantCode connect.Code
		wantErr  bool
		wantID   string
	}{
		{
			name:     "valid token injects claims",
			token:    "Bearer valid-token",
			verifier: &stubVerifier{claims: validClaims},
			wantID:   "user-1",
		},
		{
			name:     "missing authorization header",
			token:    "",
			verifier: &stubVerifier{},
			wantCode: connect.CodeUnauthenticated,
			wantErr:  true,
		},
		{
			name:     "malformed authorization header",
			token:    "InvalidFormat",
			verifier: &stubVerifier{},
			wantCode: connect.CodeUnauthenticated,
			wantErr:  true,
		},
		{
			name:     "expired token",
			token:    "Bearer expired-token",
			verifier: &stubVerifier{err: fmt.Errorf("verify token: token is expired")},
			wantCode: connect.CodeUnauthenticated,
			wantErr:  true,
		},
		{
			name:     "public procedure skips auth when no token",
			token:    "",
			verifier: &stubVerifier{},
			public:   true,
			wantCode: connect.CodeUnauthenticated, // GetCurrentUser returns unauthenticated (no claims)
			wantErr:  true,
		},
		{
			name:     "public procedure with valid token injects claims",
			token:    "Bearer valid-token",
			verifier: &stubVerifier{claims: validClaims},
			public:   true,
			wantID:   "user-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// We test via GetCurrentUser — it reads claims from context.
			svc := &stubAuthService{}
			handler := auth.NewHandler(svc)

			var opts []authz.Option
			if tt.public {
				opts = append(opts, authz.WithPublicProcedure(pbconnect.AuthServiceGetCurrentUserProcedure))
			}

			interceptor := authz.NewInterceptor(tt.verifier, opts...)
			path, h := pbconnect.NewAuthServiceHandler(handler, connect.WithInterceptors(interceptor))
			mux := http.NewServeMux()
			mux.Handle(path, h)
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			client := pbconnect.NewAuthServiceClient(srv.Client(), srv.URL)

			ctx := context.Background()
			req := connect.NewRequest(&pb.GetCurrentUserRequest{})
			if tt.token != "" {
				req.Header().Set("Authorization", tt.token)
			}

			resp, err := client.GetCurrentUser(ctx, req)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					if connectErr.Code() != tt.wantCode {
						t.Errorf("got code %v, want %v", connectErr.Code(), tt.wantCode)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Msg.GetUserId() != tt.wantID {
				t.Errorf("got user_id %q, want %q", resp.Msg.GetUserId(), tt.wantID)
			}
		})
	}
}
