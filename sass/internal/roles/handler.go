package roles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/menems/sass/pkg/httputil"
)

// roleQuerier is the read port the handler depends on.
type roleQuerier interface {
	List(ctx context.Context) ([]*Role, error)
	Get(ctx context.Context, id uuid.UUID) (*Role, error)
}

// roleMutator is the write port the handler depends on.
type roleMutator interface {
	Create(ctx context.Context, req CreateRequest) (*Role, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Role, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Handler mounts all /api/v1/roles routes.
type Handler struct {
	querier roleQuerier
	mutator roleMutator
}

// NewHandler constructs a roles Handler.
func NewHandler(q roleQuerier, m roleMutator) *Handler {
	return &Handler{querier: q, mutator: m}
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
	roles, err := h.querier.List(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list roles", nil)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"data": toRoleResponses(roles),
	})
}

// ---------------------------------------------------------------------------
// POST /roles
// ---------------------------------------------------------------------------

type permissionBody struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func toPermission(p permissionBody) Permission {
	return Permission{Resource: p.Resource, Action: p.Action}
}

type roleRequestBody struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Permissions []permissionBody `json:"permissions"`
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req roleRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required", nil)
		return
	}

	perms := make([]Permission, len(req.Permissions))
	for i, p := range req.Permissions {
		perms[i] = toPermission(p)
	}
	role, err := h.mutator.Create(r.Context(), CreateRequest{Name: req.Name, Description: req.Description, Permissions: perms})
	if err != nil {
		mapRoleError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, toRoleResponse(role))
}

// ---------------------------------------------------------------------------
// GET /roles/:id
// ---------------------------------------------------------------------------

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, r, "id")
	if !ok {
		return
	}

	role, err := h.querier.Get(r.Context(), id)
	if err != nil {
		mapRoleError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toRoleResponse(role))
}

// ---------------------------------------------------------------------------
// PUT /roles/:id
// ---------------------------------------------------------------------------

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, r, "id")
	if !ok {
		return
	}

	var req roleRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required", nil)
		return
	}

	perms := make([]Permission, len(req.Permissions))
	for i, p := range req.Permissions {
		perms[i] = toPermission(p)
	}
	role, err := h.mutator.Update(r.Context(), id, UpdateRequest{Name: req.Name, Description: req.Description, Permissions: perms})
	if err != nil {
		mapRoleError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toRoleResponse(role))
}

// ---------------------------------------------------------------------------
// DELETE /roles/:id
// ---------------------------------------------------------------------------

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParseUUID(w, r, "id")
	if !ok {
		return
	}

	if err := h.mutator.Delete(r.Context(), id); err != nil {
		mapRoleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Response shapes
// ---------------------------------------------------------------------------

type permissionResponse struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func toPermResp(p Permission) permissionResponse {
	return permissionResponse{Resource: p.Resource, Action: p.Action}
}

type roleResponse struct {
	ID          uuid.UUID            `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	IsSystem    bool                 `json:"is_system"`
	Permissions []permissionResponse `json:"permissions"`
	CreatedAt   string               `json:"created_at"`
}

func toRoleResponse(r *Role) roleResponse {
	perms := make([]permissionResponse, len(r.Permissions))
	for i, p := range r.Permissions {
		perms[i] = toPermResp(p)
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
		httputil.WriteError(w, http.StatusNotFound, "NOT_FOUND", "role not found", nil)
	case errors.Is(err, ErrNameConflict):
		httputil.WriteError(w, http.StatusConflict, "CONFLICT", "role name already in use", nil)
	case errors.Is(err, ErrSystemRole):
		httputil.WriteError(w, http.StatusConflict, "CONFLICT", "cannot delete a system role", nil)
	case errors.Is(err, ErrRoleInUse):
		httputil.WriteError(w, http.StatusConflict, "CONFLICT", "role is assigned to one or more users", nil)
	case errors.Is(err, ErrValidation):
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error", nil)
	}
}

