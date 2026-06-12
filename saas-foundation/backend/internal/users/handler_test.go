package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Router setup
// ---------------------------------------------------------------------------

// noopPerm is a requirePerm factory that always allows the request through.
func noopPerm(_, _ string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func buildRouter(t *testing.T) (*chi.Mux, *mockRepo) {
	t.Helper()
	repo := newMockRepo()
	svc := NewService(repo, bcrypt.MinCost)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		h.Mount(r, noopPerm)
	})
	return r, repo
}

func errCode(t *testing.T, body []byte) string {
	t.Helper()
	var env map[string]map[string]any
	require.NoError(t, json.Unmarshal(body, &env))
	code, _ := env["error"]["code"].(string)
	return code
}

// ---------------------------------------------------------------------------
// GET /users
// ---------------------------------------------------------------------------

func TestHandler_ListUsers(t *testing.T) {
	t.Run("200 returns paginated list", func(t *testing.T) {
		router, repo := buildRouter(t)
		repo.addUser(regularUser())
		repo.addUser(adminUser())

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&per_page=10", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		data, ok := body["data"].([]any)
		require.True(t, ok)
		assert.Len(t, data, 2)

		meta, ok := body["meta"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(2), meta["total"])
	})
}

// ---------------------------------------------------------------------------
// POST /users
// ---------------------------------------------------------------------------

func TestHandler_CreateUser(t *testing.T) {
	roleID := uuid.New()

	tests := []struct {
		name       string
		body       string
		setup      func(*mockRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "201 creates user",
			body:       fmt.Sprintf(`{"email":"new@x.com","name":"New","password":"password1","role_id":"%s"}`, roleID),
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "400 on missing fields",
			body:       `{"email":""}`,
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "400 on missing role_id",
			body:       `{"email":"a@b.com","name":"A","password":"password1"}`,
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "409 on duplicate email",
			body: fmt.Sprintf(`{"email":"dup@x.com","name":"D","password":"password1","role_id":"%s"}`, roleID),
			setup: func(r *mockRepo) {
				u := regularUser()
				u.Email = "dup@x.com"
				r.addUser(u)
			},
			wantStatus: http.StatusConflict,
			wantCode:   "CONFLICT",
		},
		{
			name:       "400 on password too short",
			body:       fmt.Sprintf(`{"email":"new@x.com","name":"N","password":"short","role_id":"%s"}`, roleID),
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, repo := buildRouter(t)
			tt.setup(repo)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, errCode(t, rec.Body.Bytes()))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GET /users/:id
// ---------------------------------------------------------------------------

func TestHandler_GetUser(t *testing.T) {
	t.Run("200 returns user", func(t *testing.T) {
		router, repo := buildRouter(t)
		u := regularUser()
		repo.addUser(u)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+u.ID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, u.Email, body["email"])
	})

	t.Run("404 for unknown id", func(t *testing.T) {
		router, _ := buildRouter(t)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+uuid.New().String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 for invalid uuid", func(t *testing.T) {
		router, _ := buildRouter(t)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/not-a-uuid", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// PUT /users/:id
// ---------------------------------------------------------------------------

func TestHandler_UpdateUser(t *testing.T) {
	roleID := uuid.New()

	t.Run("200 updates user", func(t *testing.T) {
		router, repo := buildRouter(t)
		u := regularUser()
		repo.addUser(u)

		body := fmt.Sprintf(`{"email":"updated@x.com","name":"Updated","role_id":"%s"}`, roleID)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+u.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "updated@x.com", resp["email"])
	})

	t.Run("404 for unknown user", func(t *testing.T) {
		router, _ := buildRouter(t)
		body := fmt.Sprintf(`{"email":"x@x.com","name":"X","role_id":"%s"}`, roleID)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// POST /users/:id/deactivate
// ---------------------------------------------------------------------------

func TestHandler_Deactivate(t *testing.T) {
	t.Run("200 deactivates user", func(t *testing.T) {
		router, repo := buildRouter(t)
		u := regularUser()
		repo.addUser(u)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+u.ID.String()+"/deactivate", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, false, resp["is_active"])
	})

	t.Run("409 when deactivating last admin", func(t *testing.T) {
		router, repo := buildRouter(t)
		u := adminUser()
		repo.addUser(u)
		repo.activeAdminCount = 1

		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+u.ID.String()+"/deactivate", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "CONFLICT", errCode(t, rec.Body.Bytes()))
	})
}

// ---------------------------------------------------------------------------
// POST /users/:id/reactivate
// ---------------------------------------------------------------------------

func TestHandler_Reactivate(t *testing.T) {
	t.Run("200 reactivates user", func(t *testing.T) {
		router, repo := buildRouter(t)
		u := regularUser()
		u.IsActive = false
		repo.addUser(u)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+u.ID.String()+"/reactivate", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, true, resp["is_active"])
	})
}

// ---------------------------------------------------------------------------
// PUT /users/:id/password
// ---------------------------------------------------------------------------

func TestHandler_ResetPassword(t *testing.T) {
	t.Run("204 on success", func(t *testing.T) {
		router, repo := buildRouter(t)
		u := regularUser()
		repo.addUser(u)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+u.ID.String()+"/password",
			strings.NewReader(`{"password":"newpassword123"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("400 on short password", func(t *testing.T) {
		router, repo := buildRouter(t)
		u := regularUser()
		repo.addUser(u)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+u.ID.String()+"/password",
			strings.NewReader(`{"password":"short"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec.Body.Bytes()))
	})
}
