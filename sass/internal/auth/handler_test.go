package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test setup helpers
// ---------------------------------------------------------------------------

// buildRouter creates a chi router with auth routes mounted, using an in-memory
// repository so handler tests do not need a real database.
func buildRouter(t *testing.T) (*chi.Mux, *mockRepo) {
	t.Helper()
	repo := newMockRepo()
	svc := testService(t, repo)
	h := NewHandler(svc, svc, false /* secure=false in tests */)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes (no JWT required)
		h.Mount(r)
		// /auth/me requires the user_id to be in context; in tests we set it manually
		// so no JWT middleware is applied here — matches how handler_test directly sets context.
		r.Get("/auth/me", h.Me())
	})
	return r, repo
}

// seedUser adds a user to the mock repo and returns their plain password.
func seedUser(t *testing.T, repo *mockRepo) (*User, string) {
	t.Helper()
	password := "secret123"
	u := &User{
		ID:           uuid.New(),
		Email:        "bob@example.com",
		Name:         "Bob",
		PasswordHash: hashPassword(t, password),
		IsActive:     true,
		Roles:        []Role{{ID: uuid.New(), Name: "admin"}},
		CreatedAt:    time.Now(),
	}
	repo.addUser(u)
	return u, password
}

// loginAndGetCookies performs a login request and returns the set cookies.
func loginAndGetCookies(t *testing.T, router http.Handler, email, password string) []*http.Cookie {
	t.Helper()
	body := `{"email":"` + email + `","password":"` + password + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Result().Cookies()
}

// ---------------------------------------------------------------------------
// POST /auth/login
// ---------------------------------------------------------------------------

func TestHandler_Login(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		body       string
		setupRepo  func(*mockRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name: "200 on valid credentials",
			body: `{"email":"bob@example.com","password":"secret123"}`,
			setupRepo: func(r *mockRepo) {
				u, _ := seedUser(t, r)
				_ = u
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "400 on malformed JSON",
			body:       `not json`,
			setupRepo:  func(_ *mockRepo) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "400 on missing fields",
			body:       `{"email":""}`,
			setupRepo:  func(_ *mockRepo) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "401 on wrong password",
			body:       `{"email":"bob@example.com","password":"wrong"}`,
			setupRepo:  func(r *mockRepo) { seedUser(t, r) },
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
		{
			name: "403 on deactivated account",
			body: `{"email":"bob@example.com","password":"secret123"}`,
			setupRepo: func(r *mockRepo) {
				u, _ := seedUser(t, r)
				u.IsActive = false
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "ACCOUNT_DEACTIVATED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, repo := buildRouter(t)
			tt.setupRepo(repo)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantStatus == http.StatusOK {
				// Verify cookies are set.
				cookies := rec.Result().Cookies()
				var hasAccess, hasRefresh bool
				for _, c := range cookies {
					if c.Name == cookieAccessToken {
						hasAccess = true
						assert.True(t, c.HttpOnly)
					}
					if c.Name == cookieRefreshToken {
						hasRefresh = true
						assert.True(t, c.HttpOnly)
					}
				}
				assert.True(t, hasAccess, "access_token cookie should be set")
				assert.True(t, hasRefresh, "refresh_token cookie should be set")

				// Verify response body shape.
				var body map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
				user, ok := body["user"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "bob@example.com", user["email"])
				assert.Contains(t, user, "roles")
				// Tokens must NOT appear in body.
				assert.NotContains(t, body, "access_token")
				assert.NotContains(t, body, "refresh_token")
			}

			if tt.wantCode != "" {
				var body map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
				errObj, ok := body["error"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, errObj["code"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// POST /auth/refresh
// ---------------------------------------------------------------------------

func TestHandler_Refresh(t *testing.T) {
	t.Parallel()
	t.Run("200 rotates tokens on valid refresh cookie", func(t *testing.T) {
		router, repo := buildRouter(t)
		seedUser(t, repo)

		cookies := loginAndGetCookies(t, router, "bob@example.com", "secret123")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// New cookies must be set.
		var newAccess, newRefresh string
		for _, c := range rec.Result().Cookies() {
			if c.Name == cookieAccessToken {
				newAccess = c.Value
			}
			if c.Name == cookieRefreshToken {
				newRefresh = c.Value
			}
		}
		assert.NotEmpty(t, newAccess)
		assert.NotEmpty(t, newRefresh)

		// New tokens must differ from old ones.
		for _, c := range cookies {
			if c.Name == cookieAccessToken {
				assert.NotEqual(t, c.Value, newAccess)
			}
			if c.Name == cookieRefreshToken {
				assert.NotEqual(t, c.Value, newRefresh)
			}
		}
	})

	t.Run("401 when refresh cookie is absent", func(t *testing.T) {
		router, _ := buildRouter(t)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// POST /auth/logout
// ---------------------------------------------------------------------------

func TestHandler_Logout(t *testing.T) {
	t.Parallel()
	t.Run("204 and clears cookies", func(t *testing.T) {
		router, repo := buildRouter(t)
		seedUser(t, repo)

		cookies := loginAndGetCookies(t, router, "bob@example.com", "secret123")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify cookies are cleared (MaxAge < 0).
		for _, c := range rec.Result().Cookies() {
			if c.Name == cookieAccessToken || c.Name == cookieRefreshToken {
				assert.Less(t, c.MaxAge, 0, "cookie %s should be cleared", c.Name)
			}
		}
	})

	t.Run("204 even without cookies (idempotent)", func(t *testing.T) {
		router, _ := buildRouter(t)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// GET /auth/me
// ---------------------------------------------------------------------------

func TestHandler_Me(t *testing.T) {
	t.Parallel()
	t.Run("200 returns user profile when user_id in context", func(t *testing.T) {
		router, repo := buildRouter(t)
		u, _ := seedUser(t, repo)

		// Inject the user ID into the context to simulate a JWT-authenticated request.
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		ctx := context.WithValue(req.Context(), ContextKeyUserID, u.ID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		user, ok := body["user"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, u.Email, user["email"])
		assert.Contains(t, user, "is_active")
		assert.Contains(t, user, "created_at")
	})

	t.Run("401 when no user_id in context", func(t *testing.T) {
		router, _ := buildRouter(t)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
