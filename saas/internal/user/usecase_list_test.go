package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/menems/saas/internal/user"
)

func TestService_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		store   *mockStore
		want    int
		wantErr bool
	}{
		{
			name: "success",
			store: &mockStore{
				listFn: func(_ context.Context) ([]user.User, error) {
					return []user.User{
						{ID: "1", Email: "a@example.com", Name: "A", Role: user.RoleMember},
						{ID: "2", Email: "b@example.com", Name: "B", Role: user.RoleAdmin},
					}, nil
				},
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "store error",
			store: &mockStore{
				listFn: func(_ context.Context) ([]user.User, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := user.NewService(tt.store, &mockHasher{})
			got, err := svc.List(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("List() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(got) != tt.want {
				t.Errorf("List() len = %d, want %d", len(got), tt.want)
			}
		})
	}
}
