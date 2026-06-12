package audit

import (
	"encoding/json"
	"errors"
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
// Router setup helpers
// ---------------------------------------------------------------------------

// noopPerm is a requirePerm factory that always allows the request through.
func noopPerm(_, _ string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func buildRouter(t *testing.T, repo Repository) *chi.Mux {
	t.Helper()
	svc := NewService(repo)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		h.Mount(r, noopPerm)
	})
	return r
}

func errCode(t *testing.T, body []byte) string {
	t.Helper()
	var env map[string]map[string]any
	require.NoError(t, json.Unmarshal(body, &env))
	code, _ := env["error"]["code"].(string)
	return code
}

// ---------------------------------------------------------------------------
// GET /audit-logs
// ---------------------------------------------------------------------------

func TestHandler_ListAuditLogs(t *testing.T) {
	t.Run("200 returns paginated list", func(t *testing.T) {
		repo := &mockRepo{
			entries: []*LogEntry{sampleEntry(), sampleEntry()},
			total:   2,
		}
		router := buildRouter(t, repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?page=1&per_page=10", nil)
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
		assert.Equal(t, float64(1), meta["page"])
		assert.Equal(t, float64(10), meta["per_page"])
	})

	t.Run("200 with resource_type and action filters", func(t *testing.T) {
		entry := sampleEntry()
		entry.Action = "role.update"
		entry.ResourceType = "role"
		repo := &mockRepo{entries: []*LogEntry{entry}, total: 1}
		router := buildRouter(t, repo)

		req := httptest.NewRequest(http.MethodGet,
			"/api/v1/audit-logs?resource_type=role&action=role.update", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("200 with valid actor_id filter", func(t *testing.T) {
		actorID := uuid.New()
		entry := sampleEntry()
		entry.Actor.ID = actorID
		repo := &mockRepo{entries: []*LogEntry{entry}, total: 1}
		router := buildRouter(t, repo)

		req := httptest.NewRequest(http.MethodGet,
			"/api/v1/audit-logs?actor_id="+actorID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("200 with from and to filters", func(t *testing.T) {
		repo := &mockRepo{entries: []*LogEntry{sampleEntry()}, total: 1}
		router := buildRouter(t, repo)

		req := httptest.NewRequest(http.MethodGet,
			"/api/v1/audit-logs?from=2024-01-01T00:00:00Z&to=2024-12-31T23:59:59Z", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("400 on invalid actor_id", func(t *testing.T) {
		router := buildRouter(t, &mockRepo{})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?actor_id=not-a-uuid", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec.Body.Bytes()))
	})

	t.Run("400 on invalid from timestamp", func(t *testing.T) {
		router := buildRouter(t, &mockRepo{})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?from=not-a-date", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec.Body.Bytes()))
	})

	t.Run("400 on invalid to timestamp", func(t *testing.T) {
		router := buildRouter(t, &mockRepo{})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?to=bad-timestamp", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "VALIDATION_ERROR", errCode(t, rec.Body.Bytes()))
	})

	t.Run("500 on repository error", func(t *testing.T) {
		repo := &mockRepo{errOnList: errors.New("db error")}
		router := buildRouter(t, repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "INTERNAL_ERROR", errCode(t, rec.Body.Bytes()))
	})
}

// ---------------------------------------------------------------------------
// GET /audit-logs/export
// ---------------------------------------------------------------------------

func TestHandler_ExportAuditLogs(t *testing.T) {
	t.Run("200 returns valid CSV with headers and data rows", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		resID := uuid.New()
		entry := &LogEntry{
			ID:           uuid.New(),
			Actor:        Actor{ID: uuid.New(), Name: "Jane Smith", Email: "jane@example.com"},
			Action:       "user.create",
			ResourceType: "user",
			ResourceID:   &resID,
			IPAddress:    "10.0.0.1",
			CreatedAt:    now,
		}
		repo := &mockRepo{entries: []*LogEntry{entry}}
		router := buildRouter(t, repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs/export", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
		assert.Contains(t, rec.Header().Get("Content-Disposition"), "attachment")
		assert.Contains(t, rec.Header().Get("Content-Disposition"), ".csv")

		body := rec.Body.String()
		// Header row
		assert.True(t, strings.HasPrefix(body, "id,actor_name,actor_email,action"))
		// Data row
		assert.Contains(t, body, "jane@example.com")
		assert.Contains(t, body, "user.create")
		assert.Contains(t, body, "10.0.0.1")
		assert.Contains(t, body, resID.String())
	})

	t.Run("200 returns empty CSV with header row only when no entries", func(t *testing.T) {
		repo := &mockRepo{entries: nil}
		router := buildRouter(t, repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs/export", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
		assert.Contains(t, rec.Body.String(), "id,actor_name")
	})

	t.Run("200 omits resource_id cell when nil", func(t *testing.T) {
		entry := &LogEntry{
			ID:           uuid.New(),
			Actor:        Actor{ID: uuid.New(), Name: "Bob", Email: "bob@example.com"},
			Action:       "role.delete",
			ResourceType: "role",
			ResourceID:   nil, // intentionally nil
			CreatedAt:    time.Now().UTC(),
		}
		repo := &mockRepo{entries: []*LogEntry{entry}}
		router := buildRouter(t, repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs/export", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "bob@example.com")
	})

	t.Run("500 on repository error", func(t *testing.T) {
		repo := &mockRepo{errOnExport: errors.New("db error")}
		router := buildRouter(t, repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs/export", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "INTERNAL_ERROR", errCode(t, rec.Body.Bytes()))
	})
}
