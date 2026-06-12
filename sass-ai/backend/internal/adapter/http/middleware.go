package http

import (
	"context"
	"net/http"
	"strings"

	"sassai/backend/internal/auth"
	"sassai/backend/internal/ctxkey"
)

// RequireAuth is a middleware that validates a Bearer JWT and injects the
// user ID into the request context via ctxkey.UserID.
func RequireAuth(tokens auth.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			userID, _, err := tokens.Validate(tokenStr)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxkey.UserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
