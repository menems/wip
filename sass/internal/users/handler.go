package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/menems/sass/pkg/httputil"
)


// ---------------------------------------------------------------------------
// UserService interfaces — consumer-side, defined here where they are used.
// Each covers one cohesive concern and has ≤3 methods.
// ---------------------------------------------------------------------------

// userQuerier is the read port the handler depends on.
type userQuerier interface {
	List(ctx context.Context, filter UserFilter) ([]*User, int, error)
	Get(ctx context.Context, id uuid.UUID) (*User, error)
}

// userMutator is the write port the handler depends on.
type userMutator interface {
	Create(ctx context.Context, req CreateRequest) (*User, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*User, error)
}

// userActioner is the lifecycle port the handler depends on.
type userActioner interface {
	SetActive(ctx context.Context, id uuid.UUID, active bool) (*User, error)
	ChangePassword(ctx context.Context, id uuid.UUID, oldPwd, newPwd string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler mounts all /api/v1/users routes.
type Handler struct {
	querier  userQuerier
	mutator  userMutator
	actioner userActioner
}

// NewHandler constructs a users Handler.
func NewHandler(q userQuerier, m userMutator, a userActioner) *Handler {
	return &Handler{querier: q, mutator: m, actioner: a}
}

// Mount registers all user routes on the given router.
// All routes inside this group must already have JWTAuth applied by the caller.
// RBAC is enforced per-route using the injected requirePerm factory.
func (h *Handler) Mount(r chi.Router, requirePerm func(resource, action string) func(http.Handler) http.Handler) {
	r.With(requirePerm("users", "read")).Get("/users", h.handleList)
	r.With(requirePerm("users", "write")).Post("/users", h.handleCreate)
	r.With(requirePerm("users", "read")).Get("/users/{id}", h.handleGet)
	r.With(requirePerm("users", "write")).Put("/users/{id}", h.handleUpdate)
	r.With(requirePerm("users", "write")).Patch("/users/{id}/status", h.handleSetActive)
	r.With(requirePerm("users", "write")).Put("/users/{id}/password", h.handleChangePassword)
	r.With(requirePerm("users", "delete")).Delete("/users/{id}", h.handleDelete)
}

// ---------------------------------------------------------------------------
// GET /users
// ---------------------------------------------------------------------------

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	filter := UserFilter{
		Search:  q.Get("search"),
		Page:    page,
		PerPage: perPage,
		SortBy:  q.Get("sort_by"),
		SortDir: q.Get("sort_dir"),
	}

	users, total, err := h.querier.List(r.Context(), filter)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list users", nil)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"data":  toUserResponses(users),
		"total": total,
	})
}

// ---------------------------------------------------------------------------
// POST /users
// ---------------------------------------------------------------------------

type createUserBody struct {
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Password string    `json:"password"`
	RoleID   uuid.UUID `json:"role_id"`
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body createUserBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if body.Email == "" || body.Name == "" || body.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "email, name, and password are required", nil)
		return
	}

	user, err := h.mutator.Create(r.Context(), CreateRequest(body))
	if err != nil {
		mapUserError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, toUserResponse(user))
}

// ---------------------------------------------------------------------------
// GET /users/{id}
// ---------------------------------------------------------------------------

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, r, "id")
	if !ok {
		return
	}

	user, err := h.querier.Get(r.Context(), id)
	if err != nil {
		mapUserError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

// ---------------------------------------------------------------------------
// PUT /users/{id}
// ---------------------------------------------------------------------------

type updateUserBody struct {
	Name   string    `json:"name"`
	Email  string    `json:"email"`
	RoleID uuid.UUID `json:"role_id"`
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, r, "id")
	if !ok {
		return
	}

	var body updateUserBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if body.Name == "" || body.Email == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and email are required", nil)
		return
	}

	user, err := h.mutator.Update(r.Context(), id, UpdateRequest(body))
	if err != nil {
		mapUserError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

// ---------------------------------------------------------------------------
// PATCH /users/{id}/status
// ---------------------------------------------------------------------------

type setActiveBody struct {
	IsActive bool `json:"is_active"`
}

func (h *Handler) handleSetActive(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, r, "id")
	if !ok {
		return
	}

	var body setActiveBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	user, err := h.actioner.SetActive(r.Context(), id, body.IsActive)
	if err != nil {
		mapUserError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

// ---------------------------------------------------------------------------
// PUT /users/{id}/password
// ---------------------------------------------------------------------------

type changePasswordBody struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, r, "id")
	if !ok {
		return
	}

	var body changePasswordBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if body.OldPassword == "" || body.NewPassword == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "old_password and new_password are required", nil)
		return
	}

	if err := h.actioner.ChangePassword(r.Context(), id, body.OldPassword, body.NewPassword); err != nil {
		mapUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// DELETE /users/{id}
// ---------------------------------------------------------------------------

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, r, "id")
	if !ok {
		return
	}

	if err := h.actioner.Delete(r.Context(), id); err != nil {
		mapUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Response shapes
// ---------------------------------------------------------------------------

type roleResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	IsSystem bool      `json:"is_system"`
}

func toRoleResp(r Role) roleResponse {
	return roleResponse{ID: r.ID, Name: r.Name, IsSystem: r.IsSystem}
}

type userResponse struct {
	ID        uuid.UUID      `json:"id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	IsActive  bool           `json:"is_active"`
	Roles     []roleResponse `json:"roles"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

func toUserResponse(u *User) userResponse {
	roles := make([]roleResponse, len(u.Roles))
	for i, r := range u.Roles {
		roles[i] = toRoleResp(r)
	}
	return userResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		IsActive:  u.IsActive,
		Roles:     roles,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toUserResponses(users []*User) []userResponse {
	out := make([]userResponse, len(users))
	for i, u := range users {
		out[i] = toUserResponse(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

func mapUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found", nil)
	case errors.Is(err, ErrEmailConflict):
		httputil.WriteError(w, http.StatusConflict, "CONFLICT", "email already in use", nil)
	case errors.Is(err, ErrLastAdmin):
		httputil.WriteError(w, http.StatusConflict, "CONFLICT", "cannot remove the last active admin", nil)
	case errors.Is(err, ErrInvalidPassword):
		httputil.WriteError(w, http.StatusUnprocessableEntity, "INVALID_PASSWORD", "current password is incorrect", nil)
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error", nil)
	}
}

