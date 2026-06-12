package auth_test

import (
	"testing"
	"time"

	"github.com/menems/saas/internal/auth"
)

func TestClaims_ZeroValue(t *testing.T) {
	t.Parallel()

	var c auth.Claims

	if c.UserID != "" {
		t.Errorf("zero Claims.UserID = %q, want empty", c.UserID)
	}
	if c.Email != "" {
		t.Errorf("zero Claims.Email = %q, want empty", c.Email)
	}
	if c.Role != "" {
		t.Errorf("zero Claims.Role = %q, want empty", c.Role)
	}
	if !c.ExpiresAt.IsZero() {
		t.Errorf("zero Claims.ExpiresAt = %v, want zero", c.ExpiresAt)
	}
	if !c.IssuedAt.IsZero() {
		t.Errorf("zero Claims.IssuedAt = %v, want zero", c.IssuedAt)
	}
}

func TestClaims_FieldAssignment(t *testing.T) {
	t.Parallel()

	now := time.Now()
	c := auth.Claims{
		UserID:    "user-123",
		Email:     "alice@example.com",
		Role:      "admin",
		ExpiresAt: now.Add(time.Hour),
		IssuedAt:  now,
	}

	if c.UserID != "user-123" {
		t.Errorf("Claims.UserID = %q, want %q", c.UserID, "user-123")
	}
	if c.Email != "alice@example.com" {
		t.Errorf("Claims.Email = %q, want %q", c.Email, "alice@example.com")
	}
	if c.Role != "admin" {
		t.Errorf("Claims.Role = %q, want %q", c.Role, "admin")
	}
	if !c.ExpiresAt.After(now) {
		t.Error("Claims.ExpiresAt should be after now")
	}
	if !c.IssuedAt.Equal(now) {
		t.Error("Claims.IssuedAt should equal now")
	}
}
