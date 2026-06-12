package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"sassai/backend/internal/auth"
	"sassai/backend/internal/user"
)

// AuthHandler is the HTTP adapter for authentication use cases.
type AuthHandler struct {
	svc auth.Service
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(svc auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register handles POST /api/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Register(r.Context(), auth.RegisterParams{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		switch {
		case errors.Is(err, user.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email already in use")
		case err.Error() == "email is required" || err.Error() == "password must be at least 8 characters":
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Login(r.Context(), auth.LoginParams{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
