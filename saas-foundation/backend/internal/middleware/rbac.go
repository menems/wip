package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/your-org/saas-foundation/backend/internal/auth"
)

// Permission represents a single allowed (resource, action) pair.
type Permission struct {
	Resource string
	Action   string
}

// PermissionLoader is the port the RBAC middleware depends on.
// Implementations load the effective permission set for a given user.
type PermissionLoader interface {
	// LoadPermissions returns the union of all permissions across all roles
	// assigned to the user identified by userID.
	LoadPermissions(ctx context.Context, userID uuid.UUID) ([]Permission, error)
}

// RequirePermission returns middleware that enforces a single (resource, action)
// check against the authenticated user's effective permission set.
//
// The request must have already passed through JWTAuth so that
// auth.ContextKeyUserID is present in the context.
func RequirePermission(loader PermissionLoader, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(auth.ContextKeyUserID).(uuid.UUID)
			if !ok || userID == uuid.Nil {
				// JWT middleware should have caught this, but be defensive.
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
				return
			}

			permissions, err := loader.LoadPermissions(r.Context(), userID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load permissions")
				return
			}

			if !hasPermission(permissions, resource, action) {
				writeError(w, http.StatusForbidden, "FORBIDDEN",
					"you do not have permission to perform this action")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hasPermission reports whether the permission set contains the given (resource, action) pair.
func hasPermission(perms []Permission, resource, action string) bool {
	for _, p := range perms {
		if p.Resource == resource && p.Action == action {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Shared JSON error helper (used by both jwt.go and rbac.go)
// ---------------------------------------------------------------------------

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorBody{ //nolint:errcheck
		Error: errorDetail{Code: code, Message: message},
	})
}
