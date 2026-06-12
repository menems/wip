package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/menems/saas/internal/auth"
	"github.com/menems/saas/internal/user"
)

func TestService_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		email    string
		password string
		finder   *mockUserFinder
		comparer *mockPasswordComparer
		wantErr  bool
	}{
		{
			name:     "success",
			email:    "alice@example.com",
			password: "securepassword",
			finder: &mockUserFinder{
				getByEmailFn: func(_ context.Context, email string) (user.User, error) {
					return user.User{
						ID:           "user-123",
						Email:        email,
						Name:         "Alice",
						PasswordHash: "$2a$10$hashedpassword",
						Role:         user.RoleAdmin,
					}, nil
				},
			},
			comparer: &mockPasswordComparer{
				compareFn: func(_, _ string) error { return nil },
			},
			wantErr: false,
		},
		{
			name:     "user not found",
			email:    "unknown@example.com",
			password: "securepassword",
			finder: &mockUserFinder{
				getByEmailFn: func(_ context.Context, _ string) (user.User, error) {
					return user.User{}, errors.New("not found")
				},
			},
			comparer: &mockPasswordComparer{},
			wantErr:  true,
		},
		{
			name:     "wrong password",
			email:    "alice@example.com",
			password: "wrongpassword",
			finder: &mockUserFinder{
				getByEmailFn: func(_ context.Context, email string) (user.User, error) {
					return user.User{
						ID:           "user-123",
						Email:        email,
						PasswordHash: "$2a$10$hashedpassword",
						Role:         user.RoleMember,
					}, nil
				},
			},
			comparer: &mockPasswordComparer{
				compareFn: func(_, _ string) error { return errors.New("mismatch") },
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := auth.NewService(tt.finder, tt.comparer, []byte("test-secret"), time.Hour)
			token, err := svc.Login(context.Background(), tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Login() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && token == "" {
				t.Error("Login() returned empty token on success")
			}
		})
	}
}
