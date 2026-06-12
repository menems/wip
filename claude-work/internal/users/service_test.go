package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/blaz/serve/internal/users"
)

func newTestService() users.Service {
	return users.NewService(users.NewRepository())
}

func TestService_RegisterAndSignIn(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	u, err := svc.Register(ctx, "Alice", "alice@example.com", "secret")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if u.Name != "Alice" || u.Email != "alice@example.com" {
		t.Fatalf("unexpected user %+v", u)
	}
	if u.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatal("expected non-zero UUID")
	}

	token, err := svc.SignIn(ctx, "alice@example.com", "secret")
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestService_Register_Conflict(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "Alice", "alice@example.com", "secret")
	if err != nil {
		t.Fatalf("first Register: %v", err)
	}
	_, err = svc.Register(ctx, "Alice2", "alice@example.com", "other")
	if !errors.Is(err, users.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestService_SignIn_NotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.SignIn(context.Background(), "ghost@example.com", "pass")
	if !errors.Is(err, users.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestService_SignIn_BadCredentials(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _ = svc.Register(ctx, "Bob", "bob@example.com", "correct")
	_, err := svc.SignIn(ctx, "bob@example.com", "wrong")
	if !errors.Is(err, users.ErrBadCredentials) {
		t.Fatalf("want ErrBadCredentials, got %v", err)
	}
}

func TestService_FindUserByToken(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	u, _ := svc.Register(ctx, "Carol", "carol@example.com", "pass")
	token, _ := svc.SignIn(ctx, "carol@example.com", "pass")

	gotID, err := svc.FindUserByToken(ctx, token)
	if err != nil {
		t.Fatalf("FindUserByToken: %v", err)
	}
	if gotID != u.ID {
		t.Fatalf("want %v, got %v", u.ID, gotID)
	}
}

func TestService_FindUserByToken_Invalid(t *testing.T) {
	svc := newTestService()
	_, err := svc.FindUserByToken(context.Background(), "bad-token")
	if !errors.Is(err, users.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
