package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/menems/saas/internal/user"
)

func TestService_Create(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name     string
		user     user.User
		password string
		store    *mockStore
		hasher   *mockHasher
		wantErr  bool
	}{
		{
			name: "success",
			user: user.User{
				Email: "alice@example.com",
				Name:  "Alice",
				Role:  user.RoleMember,
			},
			password: "securepassword",
			store: &mockStore{
				createFn: func(_ context.Context, u user.User) (user.User, error) {
					u.CreatedAt = now
					u.UpdatedAt = now
					return u, nil
				},
			},
			hasher: &mockHasher{
				hashFn: func(_ string) (string, error) {
					return "$2a$10$hashedpassword", nil
				},
			},
			wantErr: false,
		},
		{
			name: "invalid user — empty name",
			user: user.User{
				Email: "alice@example.com",
				Name:  "",
				Role:  user.RoleMember,
			},
			password: "securepassword",
			store:    &mockStore{},
			hasher:   &mockHasher{},
			wantErr:  true,
		},
		{
			name: "invalid password — too short",
			user: user.User{
				Email: "alice@example.com",
				Name:  "Alice",
				Role:  user.RoleMember,
			},
			password: "short",
			store:    &mockStore{},
			hasher:   &mockHasher{},
			wantErr:  true,
		},
		{
			name: "hash failure",
			user: user.User{
				Email: "alice@example.com",
				Name:  "Alice",
				Role:  user.RoleMember,
			},
			password: "securepassword",
			store:    &mockStore{},
			hasher: &mockHasher{
				hashFn: func(_ string) (string, error) {
					return "", errors.New("hash error")
				},
			},
			wantErr: true,
		},
		{
			name: "store failure",
			user: user.User{
				Email: "alice@example.com",
				Name:  "Alice",
				Role:  user.RoleMember,
			},
			password: "securepassword",
			store: &mockStore{
				createFn: func(_ context.Context, _ user.User) (user.User, error) {
					return user.User{}, errors.New("db error")
				},
			},
			hasher: &mockHasher{
				hashFn: func(_ string) (string, error) {
					return "$2a$10$hashedpassword", nil
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := user.NewService(tt.store, tt.hasher)
			got, err := svc.Create(context.Background(), tt.user, tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.Email != tt.user.Email {
					t.Errorf("Create() email = %q, want %q", got.Email, tt.user.Email)
				}
				if got.PasswordHash == "" {
					t.Error("Create() password hash should be set")
				}
				if got.ID == "" {
					t.Error("Create() ID should be set")
				}
			}
		})
	}
}
