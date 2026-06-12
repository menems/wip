package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/menems/saas/internal/auth"
	"github.com/menems/saas/internal/user"
)

func TestService_VerifyToken(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret")
	finder := &mockUserFinder{
		getByEmailFn: func(_ context.Context, email string) (user.User, error) {
			return user.User{
				ID:           "user-123",
				Email:        email,
				Name:         "Alice",
				PasswordHash: "$2a$10$hashedpassword",
				Role:         user.RoleAdmin,
			}, nil
		},
	}
	comparer := &mockPasswordComparer{
		compareFn: func(_, _ string) error { return nil },
	}

	t.Run("valid token round-trip", func(t *testing.T) {
		t.Parallel()
		svc := auth.NewService(finder, comparer, secret, time.Hour)

		token, err := svc.Login(context.Background(), "alice@example.com", "securepassword")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		claims, err := svc.VerifyToken(context.Background(), token)
		if err != nil {
			t.Fatalf("VerifyToken() error = %v", err)
		}
		if claims.UserID != "user-123" {
			t.Errorf("claims.UserID = %q, want %q", claims.UserID, "user-123")
		}
		if claims.Email != "alice@example.com" {
			t.Errorf("claims.Email = %q, want %q", claims.Email, "alice@example.com")
		}
		if claims.Role != "admin" {
			t.Errorf("claims.Role = %q, want %q", claims.Role, "admin")
		}
		if claims.ExpiresAt.Before(time.Now()) {
			t.Error("claims.ExpiresAt should be in the future")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		t.Parallel()
		svc := auth.NewService(finder, comparer, secret, time.Hour)

		_, err := svc.VerifyToken(context.Background(), "invalid.token.string")
		if err == nil {
			t.Error("VerifyToken() expected error for invalid token")
		}
	})

	t.Run("wrong signing key", func(t *testing.T) {
		t.Parallel()
		svc1 := auth.NewService(finder, comparer, []byte("key-one"), time.Hour)
		svc2 := auth.NewService(finder, comparer, []byte("key-two"), time.Hour)

		token, err := svc1.Login(context.Background(), "alice@example.com", "securepassword")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		_, err = svc2.VerifyToken(context.Background(), token)
		if err == nil {
			t.Error("VerifyToken() expected error for wrong signing key")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		t.Parallel()
		// Use a negative duration to create an already-expired token.
		svc := auth.NewService(finder, comparer, secret, -time.Hour)

		token, err := svc.Login(context.Background(), "alice@example.com", "securepassword")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		_, err = svc.VerifyToken(context.Background(), token)
		if err == nil {
			t.Error("VerifyToken() expected error for expired token")
		}
	})
}
