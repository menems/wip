package user

import (
	"errors"
	"fmt"
	"net/mail"
	"time"
)

// Sentinel errors for domain operations.
var (
	ErrNotFound         = errors.New("user not found")
	ErrConflict         = errors.New("user already exists")
	ErrValidation       = errors.New("user validation failed")
	ErrPermissionDenied = errors.New("permission denied")
)

// Role represents a user's authorization level.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// User is the core domain type — no struct tags.
type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Validate checks that the user's fields satisfy domain rules.
// It does NOT validate password (plaintext is never stored on User).
func (u User) Validate() error {
	if u.Name == "" {
		return errors.New("validate user: name is required")
	}
	if err := ValidateEmail(u.Email); err != nil {
		return fmt.Errorf("validate user: %w", err)
	}
	if err := ValidateRole(u.Role); err != nil {
		return fmt.Errorf("validate user: %w", err)
	}
	return nil
}

// ValidateEmail checks that email is a valid RFC 5322 address.
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("validate email: email is required")
	}
	a, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("validate email: %w", err)
	}
	// mail.ParseAddress accepts "Alice <alice@example.com>" — reject display names.
	if a.Address != email {
		return errors.New("validate email: must be a plain address without display name")
	}
	return nil
}

// ValidatePassword checks the plaintext password meets minimum length.
func ValidatePassword(password string) error {
	const minLen = 8
	if len(password) < minLen {
		return fmt.Errorf("validate password: must be at least %d characters", minLen)
	}
	return nil
}

// ValidateRole checks that the role is a known value.
func ValidateRole(r Role) error {
	switch r {
	case RoleAdmin, RoleMember:
		return nil
	default:
		return fmt.Errorf("validate role: unknown role %q", r)
	}
}
