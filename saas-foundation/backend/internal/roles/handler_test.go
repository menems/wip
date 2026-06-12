package roles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Router setup
// ---------------------------------------------------------------------------

func noopPerm(_, _ string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func buildRouter(t *testing.T) (*chi.Mux, *mockRepo) {
	t.Helper()
	repo := newMockRepo()
	svc := NewService(repo)
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
// GET /roles
// ---------------------------------------------------------------------------

func TestHandler_ListRoles(t *testing.T) {
	t.Run("200 returns all roles with permissions", func(t *testing.T) {
		router, repo := buildRouter(t)
		repo.addRole(adminRole())
		repo.addRole(viewerRole())

		req := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		data, ok := body["data"].([]any)
		require.True(t, ok)
		assert.Len(t, data, 2)

		// Each role should have permissions array
		first := data[0].(map[string]any)
		assert.Contains(t, first, "permissions")
		assert.Contains(t, first, "is_system")
	})
}

// ---------------------------------------------------------------------------
// POST /roles
// ---------------------------------------------------------------------------

func TestHandler_CreateRole(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mockRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "201 creates role with valid permissions",
			body:       `{"name":"editor","permissions":[{"resource":"users","action":"read"}]}`,
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "201 creates role with empty permissions",
			body:       `{"name":"noop"}`,
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "400 on missing name",
			body:       `{"permissions":[]}`,
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "400 on invalid permission resource",
			body:       `{"name":"bad","permissions":[{"resource":"invoices","action":"read"}]}`,
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "400 on invalid permission action",
			body:       `{"name":"bad","permissions":[{"resource":"audit_logs","action":"write"}]}`,
			setup:      func(_ *mockRepo) {},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "409 on duplicate name",
			body: `{"name":"viewer"}`,
			setup: func(r *mockRepo) {
				r.addRole(viewerRole())
			},
			wantStatus: http.StatusConflict,
			wantCode:   "CONFLICT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, repo := buildRouter(t)
			tt.setup(repo)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/roles", strings.NewReader(tt.body))
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
// GET /roles/:id
// ---------------------------------------------------------------------------

func TestHandler_GetRole(t *testing.T) {
	t.Run("200 returns role with permissions", func(t *testing.T) {
		router, repo := buildRouter(t)
		r := viewerRole()
		repo.addRole(r)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/roles/"+r.ID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, r.Name, body["name"])
		assert.Contains(t, body, "permissions")
	})

	t.Run("404 for unknown id", func(t *testing.T) {
		router, _ := buildRouter(t)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/roles/"+uuid.New().String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 for invalid uuid", func(t *testing.T) {
		router, _ := buildRouter(t)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/roles/not-a-uuid", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// PUT /roles/:id
// ---------------------------------------------------------------------------

func TestHandler_UpdateRole(t *testing.T) {
	t.Run("200 replaces name and permissions", func(t *testing.T) {
		router, repo := buildRouter(t)
		r := viewerRole()
		repo.addRole(r)

		body := `{"name":"super-viewer","permissions":[{"resource":"users","action":"read"},{"resource":"roles","action":"read"}]}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/roles/"+r.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "super-viewer", resp["name"])
		perms := resp["permissions"].([]any)
		assert.Len(t, perms, 2)
	})

	t.Run("200 clears permissions when list is empty", func(t *testing.T) {
		router, repo := buildRouter(t)
		r := viewerRole()
		repo.addRole(r)

		body := `{"name":"viewer","permissions":[]}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/roles/"+r.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		perms := resp["permissions"].([]any)
		assert.Len(t, perms, 0)
	})

	t.Run("404 for unknown role", func(t *testing.T) {
		router, _ := buildRouter(t)
		body := `{"name":"x","permissions":[]}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/roles/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("409 when name taken by another role", func(t *testing.T) {
		router, repo := buildRouter(t)
		r1 := viewerRole()
		r2 := &Role{ID: uuid.New(), Name: "editor", Permissions: []Permission{}}
		repo.addRole(r1)
		repo.addRole(r2)

		body := `{"name":"editor","permissions":[]}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/roles/"+r1.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// DELETE /roles/:id
// ---------------------------------------------------------------------------

func TestHandler_DeleteRole(t *testing.T) {
	t.Run("204 deletes unused non-system role", func(t *testing.T) {
		router, repo := buildRouter(t)
		r := viewerRole()
		repo.addRole(r)
		repo.userCounts[r.ID] = 0

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/roles/"+r.ID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		_, stillExists := repo.roles[r.ID]
		assert.False(t, stillExists)
	})

	t.Run("409 for system role", func(t *testing.T) {
		router, repo := buildRouter(t)
		r := adminRole()
		repo.addRole(r)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/roles/"+r.ID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "CONFLICT", errCode(t, rec.Body.Bytes()))
	})

	t.Run("409 when role has assigned users", func(t *testing.T) {
		router, repo := buildRouter(t)
		r := viewerRole()
		repo.addRole(r)
		repo.userCounts[r.ID] = 2

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/roles/"+r.ID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, "CONFLICT", errCode(t, rec.Body.Bytes()))
	})

	t.Run("404 for unknown role", func(t *testing.T) {
		router, _ := buildRouter(t)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/roles/"+uuid.New().String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
