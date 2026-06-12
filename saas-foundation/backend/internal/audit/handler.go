package audit

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler mounts all /api/v1/audit-logs routes.
type Handler struct {
	svc *Service
}

// NewHandler constructs an audit Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Mount registers the audit log routes on the given router.
// The caller must have already applied JWTAuth. RBAC is enforced per-route
// using the supplied requirePerm factory, matching the pattern used by other handlers.
func (h *Handler) Mount(r chi.Router, requirePerm func(resource, action string) func(http.Handler) http.Handler) {
	r.With(requirePerm("audit_logs", "read")).Get("/audit-logs", h.handleList)
	r.With(requirePerm("audit_logs", "read")).Get("/audit-logs/export", h.handleExport)
}

// ---------------------------------------------------------------------------
// GET /audit-logs
// ---------------------------------------------------------------------------

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseFilter(w, r)
	if !ok {
		return
	}
	filter.Page = queryInt(r, "page", 1)
	filter.PerPage = min(queryInt(r, "per_page", 25), 100)

	entries, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not retrieve audit logs", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": toLogResponses(entries),
		"meta": map[string]any{
			"page":     filter.Page,
			"per_page": filter.PerPage,
			"total":    total,
		},
	})
}

// ---------------------------------------------------------------------------
// GET /audit-logs/export
// ---------------------------------------------------------------------------

func (h *Handler) handleExport(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseFilter(w, r)
	if !ok {
		return
	}

	entries, err := h.svc.Export(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not export audit logs", nil)
		return
	}

	filename := fmt.Sprintf("audit-log-%s.csv", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	cw := csv.NewWriter(w)
	cw.Write([]string{ //nolint:errcheck
		"id", "actor_name", "actor_email", "action",
		"resource_type", "resource_id", "ip_address", "created_at",
	})

	for _, e := range entries {
		resourceID := ""
		if e.ResourceID != nil {
			resourceID = e.ResourceID.String()
		}
		cw.Write([]string{ //nolint:errcheck
			e.ID.String(),
			e.Actor.Name,
			e.Actor.Email,
			e.Action,
			e.ResourceType,
			resourceID,
			e.IPAddress,
			e.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	cw.Flush()
}

// ---------------------------------------------------------------------------
// Filter parsing
// ---------------------------------------------------------------------------

// parseFilter reads the filter query parameters shared by the list and export
// endpoints. Returns false and writes an error response on invalid input.
func parseFilter(w http.ResponseWriter, r *http.Request) (Filter, bool) {
	filter := Filter{
		ResourceType: r.URL.Query().Get("resource_type"),
		Action:       r.URL.Query().Get("action"),
		SortDir:      r.URL.Query().Get("sort_dir"),
	}

	if raw := r.URL.Query().Get("actor_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "actor_id must be a valid UUID", nil)
			return Filter{}, false
		}
		filter.ActorID = &id
	}

	if raw := r.URL.Query().Get("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "from must be an ISO8601 timestamp", nil)
			return Filter{}, false
		}
		filter.From = &t
	}

	if raw := r.URL.Query().Get("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "to must be an ISO8601 timestamp", nil)
			return Filter{}, false
		}
		filter.To = &t
	}

	return filter, true
}

// ---------------------------------------------------------------------------
// Response shapes
// ---------------------------------------------------------------------------

type logEntryResponse struct {
	ID           uuid.UUID       `json:"id"`
	Actor        actorResponse   `json:"actor"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *uuid.UUID      `json:"resource_id"`
	BeforeState  json.RawMessage `json:"before_state"`
	AfterState   json.RawMessage `json:"after_state"`
	IPAddress    string          `json:"ip_address"`
	CreatedAt    string          `json:"created_at"`
}

type actorResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

func toLogResponse(e *LogEntry) logEntryResponse {
	return logEntryResponse{
		ID:           e.ID,
		Actor:        actorResponse{ID: e.Actor.ID, Name: e.Actor.Name, Email: e.Actor.Email},
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceID:   e.ResourceID,
		BeforeState:  e.BeforeState,
		AfterState:   e.AfterState,
		IPAddress:    e.IPAddress,
		CreatedAt:    e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toLogResponses(entries []*LogEntry) []logEntryResponse {
	out := make([]logEntryResponse, len(entries))
	for i, e := range entries {
		out[i] = toLogResponse(e)
	}
	return out
}

// ---------------------------------------------------------------------------
// Helpers — each adapter owns its own copies to avoid cross-package imports.
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

type errDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, message string, details any) {
	writeJSON(w, status, map[string]any{
		"error": errDetail{Code: code, Message: message, Details: details},
	})
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}
