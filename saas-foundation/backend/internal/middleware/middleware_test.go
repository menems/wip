package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/your-org/saas-foundation/backend/internal/auth"
)

// ---------------------------------------------------------------------------
// Helpers shared by all tests
// ---------------------------------------------------------------------------

// okHandler is a trivial next handler that records whether it was called.
func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

// parseErrorCode decodes the standard error envelope and returns the code string.
func parseErrorCode(t *testing.T, body []byte) string {
	t.Helper()
	var env map[string]map[string]any
	require.NoError(t, json.Unmarshal(body, &env))
	code, _ := env["error"]["code"].(string)
	return code
}

// ---------------------------------------------------------------------------
// JWT middleware tests
// ---------------------------------------------------------------------------

// buildJWTService creates a real auth.Service backed by an in-memory mock repo
// so that we can issue real, verifiable JWTs in tests.
func buildJWTService(t *testing.T) (*auth.Service, uuid.UUID) {
	t.Helper()
	// We need to create an auth.Service. Import the internal testable parts.
	// Because service_test.go helpers are in package auth (not exported), we
	// call the public constructors directly here.
	cfg := auth.TokenConfig{
		Secret:     []byte("middleware-test-secret-at-least-32-chars"),
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 720 * time.Hour,
	}
	// Use a minimal stub repo that only needs to satisfy the interface.
	repo := &stubAuthRepo{userID: uuid.New()}
	svc := auth.NewService(repo, cfg)
	return svc, repo.userID
}

// stubAuthRepo is a minimal auth.Repository used only to obtain a valid JWT
// from a real auth.Service. Only FindUserByID is exercised by VerifyAccessToken
// (it's not called during verification; verification is pure crypto).
type stubAuthRepo struct {
	userID uuid.UUID
}

func (s *stubAuthRepo) FindUserByEmail(_ context.Context, _ string) (*auth.User, error) {
	return &auth.User{
		ID:           s.userID,
		Email:        "test@example.com",
		Name:         "Test",
		PasswordHash: "$2a$04$notreal",
		IsActive:     true,
		Roles:        []auth.Role{},
	}, nil
}
func (s *stubAuthRepo) FindUserByID(_ context.Context, _ uuid.UUID) (*auth.User, error) {
	return &auth.User{ID: s.userID, Email: "test@example.com", Name: "Test", Roles: []auth.Role{}}, nil
}
func (s *stubAuthRepo) SaveRefreshToken(_ context.Context, _ *auth.RefreshToken) error { return nil }
func (s *stubAuthRepo) FindRefreshToken(_ context.Context, _ string) (*auth.RefreshToken, error) {
	return nil, auth.ErrNotFound
}
func (s *stubAuthRepo) RevokeRefreshToken(_ context.Context, _ uuid.UUID) error { return nil }

func TestJWTAuth(t *testing.T) {
	svc, userID := buildJWTService(t)

	// Issue a real access token using the service's signing logic.
	validToken, err := svc.SignToken(userID, "test@example.com", "Test")
	require.NoError(t, err)

	tests := []struct {
		name        string
		cookie      *http.Cookie
		wantNext    bool
		wantStatus  int
		wantErrCode string
	}{
		{
			name:     "passes through and sets user_id for valid token",
			cookie:   &http.Cookie{Name: "access_token", Value: validToken},
			wantNext: true,
		},
		{
			name:        "returns 401 when cookie is absent",
			cookie:      nil,
			wantNext:    false,
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "UNAUTHORIZED",
		},
		{
			name:        "returns 401 for a tampered token",
			cookie:      &http.Cookie{Name: "access_token", Value: "bad.jwt.value"},
			wantNext:    false,
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nextCalled bool
			var capturedID uuid.UUID

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				capturedID, _ = r.Context().Value(auth.ContextKeyUserID).(uuid.UUID)
				w.WriteHeader(http.StatusOK)
			})

			handler := JWTAuth(svc)(next)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantNext, nextCalled)

			if tt.wantNext {
				assert.Equal(t, userID, capturedID, "user_id should be set in context")
			} else {
				assert.Equal(t, tt.wantStatus, rec.Code)
				assert.Equal(t, tt.wantErrCode, parseErrorCode(t, rec.Body.Bytes()))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RBAC middleware tests
// ---------------------------------------------------------------------------

// mockLoader implements PermissionLoader for tests.
type mockLoader struct {
	perms []Permission
	err   error
}

func (m *mockLoader) LoadPermissions(_ context.Context, _ uuid.UUID) ([]Permission, error) {
	return m.perms, m.err
}

func TestRequirePermission(t *testing.T) {
	userID := uuid.New()

	// Helper: build a request with the user_id already set in context (as if JWT
	// middleware ran first).
	requestWithUser := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := context.WithValue(req.Context(), auth.ContextKeyUserID, userID)
		return req.WithContext(ctx)
	}

	tests := []struct {
		name        string
		loader      PermissionLoader
		request     func() *http.Request
		resource    string
		action      string
		wantNext    bool
		wantStatus  int
		wantErrCode string
	}{
		{
			name:     "passes through when user has the required permission",
			loader:   &mockLoader{perms: []Permission{{Resource: "users", Action: "read"}}},
			request:  requestWithUser,
			resource: "users",
			action:   "read",
			wantNext: true,
		},
		{
			name:        "returns 403 when user lacks the required permission",
			loader:      &mockLoader{perms: []Permission{{Resource: "users", Action: "read"}}},
			request:     requestWithUser,
			resource:    "users",
			action:      "write",
			wantNext:    false,
			wantStatus:  http.StatusForbidden,
			wantErrCode: "FORBIDDEN",
		},
		{
			name:        "returns 403 when user has no permissions",
			loader:      &mockLoader{perms: nil},
			request:     requestWithUser,
			resource:    "roles",
			action:      "delete",
			wantNext:    false,
			wantStatus:  http.StatusForbidden,
			wantErrCode: "FORBIDDEN",
		},
		{
			name: "returns 403 for wrong resource even if action matches",
			loader: &mockLoader{perms: []Permission{
				{Resource: "roles", Action: "read"},
			}},
			request:     requestWithUser,
			resource:    "users",
			action:      "read",
			wantNext:    false,
			wantStatus:  http.StatusForbidden,
			wantErrCode: "FORBIDDEN",
		},
		{
			name:   "passes through for user with multiple roles (union of permissions)",
			loader: &mockLoader{perms: []Permission{{Resource: "users", Action: "read"}, {Resource: "roles", Action: "write"}}},
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				ctx := context.WithValue(req.Context(), auth.ContextKeyUserID, userID)
				return req.WithContext(ctx)
			},
			resource: "roles",
			action:   "write",
			wantNext: true,
		},
		{
			name:        "returns 401 when no user_id in context",
			loader:      &mockLoader{},
			request:     func() *http.Request { return httptest.NewRequest(http.MethodGet, "/test", nil) },
			resource:    "users",
			action:      "read",
			wantNext:    false,
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "UNAUTHORIZED",
		},
		{
			name:        "returns 500 when loader returns an error",
			loader:      &mockLoader{err: assert.AnError},
			request:     requestWithUser,
			resource:    "users",
			action:      "read",
			wantNext:    false,
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nextCalled bool
			handler := RequirePermission(tt.loader, tt.resource, tt.action)(okHandler(&nextCalled))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, tt.request())

			assert.Equal(t, tt.wantNext, nextCalled)

			if !tt.wantNext {
				assert.Equal(t, tt.wantStatus, rec.Code)
				assert.Equal(t, tt.wantErrCode, parseErrorCode(t, rec.Body.Bytes()))
			}
		})
	}
}
