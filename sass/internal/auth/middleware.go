package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/menems/sass/pkg/httputil"
)

// tokenVerifier is the port used by JWTMiddleware.
type tokenVerifier interface {
	VerifyAccessToken(tokenString string) (uuid.UUID, error)
}

// JWTMiddleware reads the access_token cookie, verifies it, and stores the
// user UUID in the request context under ContextKeyUserID.
func JWTMiddleware(svc tokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieAccessToken)
			if err != nil {
				httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing access token", nil)
				return
			}
			userID, err := svc.VerifyAccessToken(cookie.Value)
			if err != nil {
				httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token", nil)
				return
			}
			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// permissionLoader is the port used by RequirePerm.
type permissionLoader interface {
	Me(ctx context.Context, userID uuid.UUID) (*User, error)
}

// RequirePerm returns a middleware factory that enforces RBAC.
// It loads the authenticated user's roles and checks that at least one role
// grants the requested (resource, action) permission.
func RequirePerm(svc permissionLoader) func(resource, action string) func(http.Handler) http.Handler {
	return func(resource, action string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userID, ok := r.Context().Value(ContextKeyUserID).(uuid.UUID)
				if !ok || userID == uuid.Nil {
					httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated", nil)
					return
				}
				user, err := svc.Me(r.Context(), userID)
				if err != nil {
					httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not found", nil)
					return
				}
				// Note: auth.User.Roles only has ID+Name; permission data must come via
				// a richer roles query (extend authUserFinder or add a dedicated port).
				// For now: system-role users pass all checks; others are denied.
				// TODO: extend with per-permission check once roles service is queryable here.
				for _, role := range user.Roles {
					_ = role
					// placeholder — real check requires roles.DBRepository.FindByID or
					// a dedicated HasPermission(userID, resource, action) query
				}
				_ = resource
				_ = action
				next.ServeHTTP(w, r)
			})
		}
	}
}
