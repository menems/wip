package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/menems/saas/internal/user"
)

func TestService_Delete(t *testing.T) {
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
				deleteFn: func(_ context.Context, _ string) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "store error",
			id:   "user-123",
			store: &mockStore{
				deleteFn: func(_ context.Context, _ string) error {
					return errors.New("db error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := user.NewService(tt.store, &mockHasher{})
			err := svc.Delete(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
