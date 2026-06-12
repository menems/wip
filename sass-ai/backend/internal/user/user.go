// Package user contains the core user domain: entity, errors, and value objects.
package user

import (
	"errors"
	"time"
)

// Sentinel domain errors returned by ports and propagated to adapters.
var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already in use")
)

// User is the core domain entity.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateParams holds mutable user profile fields.
type UpdateParams struct {
	Name      string
	AvatarURL string
}
