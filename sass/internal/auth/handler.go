package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/menems/sass/pkg/httputil"
)

// loginProvider is the authentication port the handler depends on.
type loginProvider interface {
	Login(ctx context.Context, email, password string) (*LoginResult, error)
	Refresh(ctx context.Context, rawToken string) (*LoginResult, error)
	Logout(ctx context.Context, rawToken string) error
}

// meProvider is the profile port the handler depends on.
type meProvider interface {
	Me(ctx context.Context, userID uuid.UUID) (*User, error)
}

// Handler mounts all /api/v1/auth routes.
type Handler struct {
	login  loginProvider
	me     meProvider
	secure bool // whether to set Secure flag on cookies (false in dev/test)
}

// NewHandler constructs an auth Handler.
// secure should be true in production (HTTPS) and false in development.
func NewHandler(lp loginProvider, mp meProvider, secure bool) *Handler {
	return &Handler{login: lp, me: mp, secure: secure}
}

// Mount registers public auth routes (login, refresh, logout) on the given router.
// /auth/me is intentionally excluded here — it requires JWT authentication and
// is mounted separately inside the authenticated route group in main.go via Me().
func (h *Handler) Mount(r chi.Router) {
	r.Post("/auth/login", h.handleLogin)
	r.Post("/auth/refresh", h.handleRefresh)
	r.Post("/auth/logout", h.handleLogout)
}

// Me returns the http.HandlerFunc for GET /auth/me.
// Mount this inside the JWTAuth-protected route group.
func (h *Handler) Me() http.HandlerFunc {
	return h.handleMe
}

// ---------------------------------------------------------------------------
// POST /auth/login
// ---------------------------------------------------------------------------

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "malformed request body", nil)
		return
	}

	if req.Email == "" || req.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "email and password are required", nil)
		return
	}

	result, err := h.login.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password", nil)
		case errors.Is(err, ErrAccountDeactivated):
			httputil.WriteError(w, http.StatusForbidden, "ACCOUNT_DEACTIVATED", "account is deactivated", nil)
		default:
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error", nil)
		}
		return
	}

	h.setAuthCookies(w, result.AccessToken, result.RefreshToken)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"user": loginUserResponse(result.User),
	})
}

// ---------------------------------------------------------------------------
// POST /auth/refresh
// ---------------------------------------------------------------------------

func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	rawToken, err := r.Cookie("refresh_token")
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "refresh token missing", nil)
		return
	}

	result, err := h.login.Refresh(r.Context(), rawToken.Value)
	if err != nil {
		if errors.Is(err, ErrTokenInvalid) {
			h.clearAuthCookies(w)
			httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "refresh token invalid or expired", nil)
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error", nil)
		return
	}

	h.setAuthCookies(w, result.AccessToken, result.RefreshToken)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"user": loginUserResponse(result.User),
	})
}

// ---------------------------------------------------------------------------
// POST /auth/logout
// ---------------------------------------------------------------------------

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Best-effort: revoke refresh token if present; never fail the logout.
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		_ = h.login.Logout(r.Context(), cookie.Value)
	}

	h.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// GET /auth/me
// ---------------------------------------------------------------------------

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ContextKeyUserID).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated", nil)
		return
	}

	user, err := h.me.Me(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not found", nil)
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error", nil)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"user": meUserResponse(user),
	})
}

// ---------------------------------------------------------------------------
// Cookie helpers
// ---------------------------------------------------------------------------

const (
	cookieAccessToken  = "access_token"
	cookieRefreshToken = "refresh_token"
)

func (h *Handler) setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieAccessToken,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.secure,
		MaxAge:   int((15 * time.Minute).Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     cookieRefreshToken,
		Value:    refreshToken,
		Path:     "/api/v1/auth/refresh",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.secure,
		MaxAge:   int((720 * time.Hour).Seconds()),
	})
}

func (h *Handler) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieAccessToken,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.secure,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     cookieRefreshToken,
		Value:    "",
		Path:     "/api/v1/auth/refresh",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.secure,
		MaxAge:   -1,
	})
}

// ---------------------------------------------------------------------------
// Response shapes
// ---------------------------------------------------------------------------

// loginUserResponse returns the user shape used by login and refresh responses.
// Roles are returned as a plain string slice per the API spec.
func loginUserResponse(u *User) map[string]any {
	roleNames := make([]string, len(u.Roles))
	for i, r := range u.Roles {
		roleNames[i] = r.Name
	}
	return map[string]any{
		"id":    u.ID,
		"email": u.Email,
		"name":  u.Name,
		"roles": roleNames,
	}
}

// meUserResponse returns the fuller user shape used by GET /auth/me.
func meUserResponse(u *User) map[string]any {
	return map[string]any{
		"id":         u.ID,
		"email":      u.Email,
		"name":       u.Name,
		"is_active":  u.IsActive,
		"roles":      u.Roles,
		"created_at": u.CreatedAt,
	}
}

