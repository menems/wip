package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Router helper
// ---------------------------------------------------------------------------

// noopPerm is a requirePerm factory that applies no middleware (all permitted).
func noopPerm(_, _ string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func buildUsersRouter(t *testing.T) (*chi.Mux, *mockStore) {
	t.Helper()
	store := newMockStore()
	svc := newService(store)
	h := NewHandler(svc, svc, svc)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		h.Mount(r, noopPerm)
	})
	return r, store
}

func seedRegular(t *testing.T, store *mockStore) *User {
	t.Helper()
	u := regularUser(t)
	store.add(u)
	return u
}

func seedAdmin(t *testing.T, store *mockStore) *User {
	t.Helper()
	u := adminUser(t)
	store.add(u)
	return u
}

func doRequest(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// ---------------------------------------------------------------------------
// GET /users
// ---------------------------------------------------------------------------

func TestHandler_List(t *testing.T) {
	t.Parallel()
	t.Run("200 returns users array and total", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		seedRegular(t, store)
		seedAdmin(t, store)

		rec := doRequest(t, router, http.MethodGet, "/api/v1/users", nil)
		assert.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.Contains(t, body, "data")
		assert.Contains(t, body, "total")
		assert.Equal(t, float64(2), body["total"])
	})
}

// ---------------------------------------------------------------------------
// POST /users
// ---------------------------------------------------------------------------

func TestHandler_Create(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		body       any
		wantStatus int
		wantCode   string
	}{
		{
			name: "201 creates user",
			body: map[string]any{
				"email":    "new@example.com",
				"name":     "New",
				"password": "secret",
				"role_id":  uuid.New(),
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "400 on missing fields",
			body:       map[string]any{"email": ""},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "400 on malformed JSON",
			body:       nil, // will send empty body that fails decode
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _ := buildUsersRouter(t)

			var rec *httptest.ResponseRecorder
			if tt.body == nil {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString("not-json"))
				req.Header.Set("Content-Type", "application/json")
				rec = httptest.NewRecorder()
				router.ServeHTTP(rec, req)
			} else {
				rec = doRequest(t, router, http.MethodPost, "/api/v1/users", tt.body)
			}

			assert.Equal(t, tt.wantStatus, rec.Code)
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
// GET /users/{id}
// ---------------------------------------------------------------------------

func TestHandler_Get(t *testing.T) {
	t.Parallel()
	t.Run("200 returns user", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		u := seedRegular(t, store)

		rec := doRequest(t, router, http.MethodGet, "/api/v1/users/"+u.ID.String(), nil)
		assert.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.Equal(t, u.Email, body["email"])
	})

	t.Run("404 for unknown user", func(t *testing.T) {
		router, _ := buildUsersRouter(t)
		rec := doRequest(t, router, http.MethodGet, "/api/v1/users/"+uuid.New().String(), nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("400 for invalid UUID", func(t *testing.T) {
		router, _ := buildUsersRouter(t)
		rec := doRequest(t, router, http.MethodGet, "/api/v1/users/not-a-uuid", nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// PUT /users/{id}
// ---------------------------------------------------------------------------

func TestHandler_Update(t *testing.T) {
	t.Parallel()
	t.Run("200 updates user", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		u := seedRegular(t, store)

		rec := doRequest(t, router, http.MethodPut, "/api/v1/users/"+u.ID.String(), map[string]any{
			"name":    "Updated",
			"email":   "updated@example.com",
			"role_id": uuid.New(),
		})
		assert.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.Equal(t, "updated@example.com", body["email"])
	})

	t.Run("400 on missing name/email", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		u := seedRegular(t, store)

		rec := doRequest(t, router, http.MethodPut, "/api/v1/users/"+u.ID.String(), map[string]any{
			"name": "",
		})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// PATCH /users/{id}/status
// ---------------------------------------------------------------------------

func TestHandler_SetActive(t *testing.T) {
	t.Parallel()
	t.Run("200 deactivates regular user", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		u := seedRegular(t, store)

		rec := doRequest(t, router, http.MethodPatch, "/api/v1/users/"+u.ID.String()+"/status", map[string]any{
			"is_active": false,
		})
		assert.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.Equal(t, false, body["is_active"])
	})

	t.Run("409 when deactivating last admin", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		u := seedAdmin(t, store)

		rec := doRequest(t, router, http.MethodPatch, "/api/v1/users/"+u.ID.String()+"/status", map[string]any{
			"is_active": false,
		})
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// PUT /users/{id}/password
// ---------------------------------------------------------------------------

func TestHandler_ChangePassword(t *testing.T) {
	t.Parallel()
	t.Run("204 on valid old password", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		// regularUser has password "pass456"
		u := &User{
			ID:           uuid.New(),
			Email:        "dave@example.com",
			Name:         "Dave",
			PasswordHash: hashPwd(t, "oldpass"),
			IsActive:     true,
			Roles:        []Role{{ID: uuid.New(), Name: "viewer"}},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		store.add(u)

		rec := doRequest(t, router, http.MethodPut, "/api/v1/users/"+u.ID.String()+"/password", map[string]any{
			"old_password": "oldpass",
			"new_password": "newpass",
		})
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("422 on wrong old password", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		u := seedRegular(t, store)

		rec := doRequest(t, router, http.MethodPut, "/api/v1/users/"+u.ID.String()+"/password", map[string]any{
			"old_password": "wrongpass",
			"new_password": "newpass",
		})
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("400 on missing fields", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		u := seedRegular(t, store)

		rec := doRequest(t, router, http.MethodPut, "/api/v1/users/"+u.ID.String()+"/password", map[string]any{
			"old_password": "",
		})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// DELETE /users/{id}
// ---------------------------------------------------------------------------

func TestHandler_Delete(t *testing.T) {
	t.Parallel()
	t.Run("204 soft-deletes regular user", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		u := seedRegular(t, store)

		rec := doRequest(t, router, http.MethodDelete, "/api/v1/users/"+u.ID.String(), nil)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.False(t, store.byID[u.ID].IsActive)
	})

	t.Run("409 when deleting last admin", func(t *testing.T) {
		router, store := buildUsersRouter(t)
		u := seedAdmin(t, store)

		rec := doRequest(t, router, http.MethodDelete, "/api/v1/users/"+u.ID.String(), nil)
		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("404 for unknown user", func(t *testing.T) {
		router, _ := buildUsersRouter(t)
		rec := doRequest(t, router, http.MethodDelete, "/api/v1/users/"+uuid.New().String(), nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
