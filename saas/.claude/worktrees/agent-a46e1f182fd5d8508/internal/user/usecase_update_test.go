package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/menems/saas/internal/user"
)

func TestService_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		user    user.User
		store   *mockStore
		wantErr bool
	}{
		{
			name: "success",
			user: user.User{
				ID:    "user-123",
				Email: "alice@example.com",
				Name:  "Alice Updated",
				Role:  user.RoleMember,
			},
			store: &mockStore{
				updateFn: func(_ context.Context, u user.User) (user.User, error) {
					return u, nil
				},
			},
			wantErr: false,
		},
		{
			name: "invalid user — empty name",
			user: user.User{
				ID:    "user-123",
				Email: "alice@example.com",
				Name:  "",
				Role:  user.RoleMember,
			},
			store:   &mockStore{},
			wantErr: true,
		},
		{
			name: "store error",
			user: user.User{
				ID:    "user-123",
				Email: "alice@example.com",
				Name:  "Alice",
				Role:  user.RoleMember,
			},
			store: &mockStore{
				updateFn: func(_ context.Context, _ user.User) (user.User, error) {
					return user.User{}, errors.New("db error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := user.NewService(tt.store, &mockHasher{})
			got, err := svc.Update(context.Background(), tt.user)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got.Name != tt.user.Name {
				t.Errorf("Update() name = %q, want %q", got.Name, tt.user.Name)
			}
		})
	}
}
