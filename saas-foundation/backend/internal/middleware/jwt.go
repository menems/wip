// Package middleware provides HTTP middleware for JWT authentication and RBAC
// permission enforcement. It sits at the adapter layer and depends on the auth
// domain package for token verification and context keys.
package middleware

import (
	"context"
	"net/http"

	"github.com/your-org/saas-foundation/backend/internal/auth"
)

// JWTAuth returns middleware that validates the access_token httpOnly cookie.
//
// On success it stores the authenticated user's UUID in the request context
// under auth.ContextKeyUserID. On failure it responds with 401 UNAUTHORIZED.
func JWTAuth(svc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("access_token")
			if err != nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "access token missing")
				return
			}

			userID, err := svc.VerifyAccessToken(cookie.Value)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "access token invalid or expired")
				return
			}

			ctx := context.WithValue(r.Context(), auth.ContextKeyUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
