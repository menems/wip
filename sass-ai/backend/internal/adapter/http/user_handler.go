package http

import (
	"encoding/json"
	"net/http"

	"sassai/backend/internal/ctxkey"
	"sassai/backend/internal/user"
)

// UserHandler is the HTTP adapter for user use cases.
type UserHandler struct {
	svc user.Service
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(svc user.Service) *UserHandler {
	return &UserHandler{svc: svc}
}

// GetMe handles GET /api/users/me.
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := ctxkey.GetUserID(r.Context())
	u, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// UpdateMe handles PATCH /api/users/me.
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := ctxkey.GetUserID(r.Context())

	var req struct {
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.svc.Update(r.Context(), userID, user.UpdateParams{
		Name:      req.Name,
		AvatarURL: req.AvatarURL,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, u)
}
