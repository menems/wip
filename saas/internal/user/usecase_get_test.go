package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/menems/saas/internal/user"
)

func TestService_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		store   *mockStore
		wantErr bool
	}{
		{
			name: "success",
			id:   "user-123",
			store: &mockStore{
				getByIDFn: func(_ context.Context, id string) (user.User, error) {
					return user.User{ID: id, Email: "alice@example.com", Name: "Alice", Role: user.RoleMember}, nil
				},
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   "missing",
			store: &mockStore{
				getByIDFn: func(_ context.Context, _ string) (user.User, error) {
					return user.User{}, errors.New("not found")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := user.NewService(tt.store, &mockHasher{})
			got, err := svc.Get(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got.ID != tt.id {
				t.Errorf("Get() ID = %q, want %q", got.ID, tt.id)
			}
		})
	}
}
