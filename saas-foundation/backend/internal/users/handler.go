package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler mounts all /api/v1/users routes.
type Handler struct {
	svc UserService
}

// NewHandler constructs a users Handler.
func NewHandler(svc UserService) *Handler {
	return &Handler{svc: svc}
}

// Mount registers all user routes on the given router.
// All routes inside this group must already have JWTAuth applied by the caller.
// RBAC is enforced per-route using chi middleware.With().
func (h *Handler) Mount(r chi.Router, requirePerm func(resource, action string) func(http.Handler) http.Handler) {
	r.With(requirePerm("users", "read")).Get("/users", h.handleList)
	r.With(requirePerm("users", "write")).Post("/users", h.handleCreate)
	r.With(requirePerm("users", "read")).Get("/users/{id}", h.handleGet)
	r.With(requirePerm("users", "write")).Put("/users/{id}", h.handleUpdate)
	r.With(requirePerm("users", "delete")).Post("/users/{id}/deactivate", h.handleDeactivate)
	r.With(requirePerm("users", "write")).Post("/users/{id}/reactivate", h.handleReactivate)
	r.With(requirePerm("users", "write")).Put("/users/{id}/password", h.handleResetPassword)
}

// ---------------------------------------------------------------------------
// GET /users
// ---------------------------------------------------------------------------

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	filter := UserFilter{
		Search:  r.URL.Query().Get("search"),
		SortBy:  r.URL.Query().Get("sort_by"),
		SortDir: r.URL.Query().Get("sort_dir"),
		Page:    queryInt(r, "page", 1),
		PerPage: min(queryInt(r, "per_page", 25), 100),
	}

	users, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list users", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": newUserResponses(users),
		"meta": map[string]any{
			"page":     filter.Page,
			"per_page": filter.PerPage,
			"total":    total,
		},
	})
}

// ---------------------------------------------------------------------------
// POST /users
// ---------------------------------------------------------------------------

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string    `json:"email"`
		Name     string    `json:"name"`
		Password string    `json:"password"`
		RoleID   uuid.UUID `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Name) == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "email, name, and password are required", nil)
		return
	}
	if req.RoleID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "role_id is required", nil)
		return
	}

	user, err := h.svc.Create(r.Context(), CreateParams{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
		RoleID:   req.RoleID,
	})
	if err != nil {
		mapUserError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newUserResponse(user))
}

// ---------------------------------------------------------------------------
// GET /users/:id
// ---------------------------------------------------------------------------

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	user, err := h.svc.Get(r.Context(), id)
	if err != nil {
		mapUserError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newUserResponse(user))
}

// ---------------------------------------------------------------------------
// PUT /users/:id
// ---------------------------------------------------------------------------

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	var req struct {
		Email  string    `json:"email"`
		Name   string    `json:"name"`
		RoleID uuid.UUID `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "email and name are required", nil)
		return
	}
	if req.RoleID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "role_id is required", nil)
		return
	}

	user, err := h.svc.Update(r.Context(), id, UpdateParams{
		Email:  req.Email,
		Name:   req.Name,
		RoleID: req.RoleID,
	})
	if err != nil {
		mapUserError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newUserResponse(user))
}

// ---------------------------------------------------------------------------
// POST /users/:id/deactivate
// ---------------------------------------------------------------------------

func (h *Handler) handleDeactivate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	user, err := h.svc.Deactivate(r.Context(), id)
	if err != nil {
		mapUserError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newUserResponse(user))
}

// ---------------------------------------------------------------------------
// POST /users/:id/reactivate
// ---------------------------------------------------------------------------

func (h *Handler) handleReactivate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	user, err := h.svc.Reactivate(r.Context(), id)
	if err != nil {
		mapUserError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newUserResponse(user))
}

// ---------------------------------------------------------------------------
// PUT /users/:id/password
// ---------------------------------------------------------------------------

func (h *Handler) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if err := h.svc.ResetPassword(r.Context(), id, req.Password); err != nil {
		mapUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Response shapes
// ---------------------------------------------------------------------------

// userResponse is the JSON representation of a User returned by the API.
type userResponse struct {
	ID        string         `json:"id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	IsActive  bool           `json:"is_active"`
	Roles     []roleResponse `json:"roles"`
	CreatedAt string         `json:"created_at"`
}

// roleResponse is the JSON representation of a Role embedded in userResponse.
type roleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// newUserResponse converts a domain User to a userResponse.
func newUserResponse(u *User) userResponse {
	roles := make([]roleResponse, len(u.Roles))
	for i, r := range u.Roles {
		roles[i] = roleResponse{ID: r.ID.String(), Name: r.Name}
	}
	return userResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		IsActive:  u.IsActive,
		Roles:     roles,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// newUserResponses converts a slice of domain Users to a slice of userResponse.
func newUserResponses(users []*User) []userResponse {
	out := make([]userResponse, len(users))
	for i, u := range users {
		out[i] = newUserResponse(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

func mapUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found", nil)
	case errors.Is(err, ErrEmailConflict):
		writeError(w, http.StatusConflict, "CONFLICT", "email already in use", nil)
	case errors.Is(err, ErrLastAdmin):
		writeError(w, http.StatusConflict, "CONFLICT", "cannot deactivate the last active admin", nil)
	case errors.Is(err, ErrValidation):
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error", nil)
	}
}

// ---------------------------------------------------------------------------
// Helpers
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
