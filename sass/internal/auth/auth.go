// Package auth implements JWT issuance, validation, and refresh token rotation.
// It follows hexagonal architecture: domain types and port interfaces live here;
// the application layer (service.go) depends only on these interfaces;
// adapters (repository.go, handler.go) implement and consume them.
package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors that callers must handle.
var (
	// ErrInvalidCredentials is returned when email or password do not match.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrAccountDeactivated is returned when the user exists but is_active=false.
	ErrAccountDeactivated = errors.New("account deactivated")

	// ErrTokenInvalid is returned when a refresh token is missing, expired, or revoked.
	ErrTokenInvalid = errors.New("refresh token invalid")

	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")
)

// Role is a slim view of a role as needed by the auth layer.
type Role struct {
	ID   uuid.UUID
	Name string
}

// User is the auth layer's view of a user record including their assigned roles.
type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	IsActive     bool
	Roles        []Role
	CreatedAt    time.Time
}

// RefreshToken represents a stored refresh token record.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// IsValid reports whether the token has not expired and has not been revoked.
func (rt *RefreshToken) IsValid() bool {
	return rt.RevokedAt == nil && time.Now().Before(rt.ExpiresAt)
}

// LoginResult is returned by a successful login or token refresh.
type LoginResult struct {
	User         *User
	AccessToken  string
	RefreshToken string // raw opaque token — goes in the cookie
}

// TokenConfig holds the signing secret and TTL values needed to issue JWTs.
type TokenConfig struct {
	Secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// ContextKey is the type used for request-context keys in this package.
type ContextKey string

// ContextKeyUserID is the request-context key that holds the authenticated
// user's UUID. Set by the JWT middleware; read by handlers and RBAC middleware.
const ContextKeyUserID ContextKey = "user_id"
