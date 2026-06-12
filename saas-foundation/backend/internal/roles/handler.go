package roles

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler mounts all /api/v1/roles routes.
type Handler struct {
	svc *Service
}

// NewHandler constructs a roles Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Mount registers all role routes on the given router.
// All routes inside this group must already have JWTAuth applied by the caller.
// RBAC is enforced per-route using the injected requirePerm factory.
func (h *Handler) Mount(r chi.Router, requirePerm func(resource, action string) func(http.Handler) http.Handler) {
	r.With(requirePerm("roles", "read")).Get("/roles", h.handleList)
	r.With(requirePerm("roles", "write")).Post("/roles", h.handleCreate)
	r.With(requirePerm("roles", "read")).Get("/roles/{id}", h.handleGet)
	r.With(requirePerm("roles", "write")).Put("/roles/{id}", h.handleUpdate)
	r.With(requirePerm("roles", "delete")).Delete("/roles/{id}", h.handleDelete)
}

// ---------------------------------------------------------------------------
// GET /roles
// ---------------------------------------------------------------------------

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list roles", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": toRoleResponses(roles),
	})
}

// ---------------------------------------------------------------------------
// POST /roles
// ---------------------------------------------------------------------------

type roleRequestBody struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req roleRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required", nil)
		return
	}

	role, err := h.svc.Create(r.Context(), CreateRequest{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	})
	if err != nil {
		mapRoleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toRoleResponse(role))
}

// ---------------------------------------------------------------------------
// GET /roles/:id
// ---------------------------------------------------------------------------

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	role, err := h.svc.Get(r.Context(), id)
	if err != nil {
		mapRoleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toRoleResponse(role))
}

// ---------------------------------------------------------------------------
// PUT /roles/:id
// ---------------------------------------------------------------------------

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	var req roleRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required", nil)
		return
	}

	role, err := h.svc.Update(r.Context(), id, UpdateRequest{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	})
	if err != nil {
		mapRoleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toRoleResponse(role))
}

// ---------------------------------------------------------------------------
// DELETE /roles/:id
// ---------------------------------------------------------------------------

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		mapRoleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Response shapes
// ---------------------------------------------------------------------------

type roleResponse struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsSystem    bool         `json:"is_system"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   string       `json:"created_at"`
}

func toRoleResponse(r *Role) roleResponse {
	perms := r.Permissions
	if perms == nil {
		perms = []Permission{}
	}
	return roleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		Permissions: perms,
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toRoleResponses(roles []*Role) []roleResponse {
	out := make([]roleResponse, len(roles))
	for i, r := range roles {
		out[i] = toRoleResponse(r)
	}
	return out
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

func mapRoleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "role not found", nil)
	case errors.Is(err, ErrNameConflict):
		writeError(w, http.StatusConflict, "CONFLICT", "role name already in use", nil)
	case errors.Is(err, ErrSystemRole):
		writeError(w, http.StatusConflict, "CONFLICT", "cannot delete a system role", nil)
	case errors.Is(err, ErrRoleInUse):
		writeError(w, http.StatusConflict, "CONFLICT", "role is assigned to one or more users", nil)
	case errors.Is(err, ErrValidation):
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error", nil)
	}
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

func parseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id format", nil)
		return uuid.Nil, false
	}
	return id, true
}

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
