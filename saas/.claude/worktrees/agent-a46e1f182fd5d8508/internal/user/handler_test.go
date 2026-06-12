package user_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/menems/saas/gen/api/v1/pb"
	"github.com/menems/saas/gen/api/v1/pb/pbconnect"
	"github.com/menems/saas/internal/user"
	"github.com/menems/saas/pkg/authz"
)

// claimsInjector is a test interceptor that injects auth claims into the server-side context.
type claimsInjector struct {
	claims *authz.Claims
}

func (ci *claimsInjector) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if ci.claims != nil {
			ctx = authz.ContextWithClaims(ctx, *ci.claims)
		}
		return next(ctx, req)
	}
}

func (ci *claimsInjector) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (ci *claimsInjector) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// mockUserService implements user.UserService for handler tests.
type mockUserService struct {
	createFn func(ctx context.Context, u user.User, password string) (user.User, error)
	listFn   func(ctx context.Context) ([]user.User, error)
	getFn    func(ctx context.Context, id string) (user.User, error)
	updateFn func(ctx context.Context, u user.User) (user.User, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockUserService) Create(ctx context.Context, u user.User, password string) (user.User, error) {
	return m.createFn(ctx, u, password)
}

func (m *mockUserService) List(ctx context.Context) ([]user.User, error) {
	return m.listFn(ctx)
}

func (m *mockUserService) Get(ctx context.Context, id string) (user.User, error) {
	return m.getFn(ctx, id)
}

func (m *mockUserService) Update(ctx context.Context, u user.User) (user.User, error) {
	return m.updateFn(ctx, u)
}

func (m *mockUserService) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

func adminClaims() *authz.Claims {
	return &authz.Claims{UserID: "admin-1", Email: "admin@example.com", Role: "admin"}
}

func memberClaims() *authz.Claims {
	return &authz.Claims{UserID: "member-1", Email: "member@example.com", Role: "member"}
}

func newTestServer(t *testing.T, svc user.UserService, claims *authz.Claims) pbconnect.UserServiceClient {
	t.Helper()
	injector := &claimsInjector{claims: claims}
	handler := user.NewHandler(svc)
	path, h := pbconnect.NewUserServiceHandler(handler, connect.WithInterceptors(injector))
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return pbconnect.NewUserServiceClient(srv.Client(), srv.URL)
}

func assertConnectCode(t *testing.T, err error, wantCode connect.Code) {
	t.Helper()
	var connectErr *connect.Error
	if errors.As(err, &connectErr) && connectErr.Code() != wantCode {
		t.Errorf("got code %v, want %v", connectErr.Code(), wantCode)
	}
}

func TestHandler_CreateUser(t *testing.T) {
	t.Parallel()

	okSvc := &mockUserService{
		createFn: func(_ context.Context, u user.User, _ string) (user.User, error) {
			u.ID = "new-id"
			return u, nil
		},
	}

	tests := []struct {
		name     string
		svc      user.UserService
		claims   *authz.Claims
		req      *pb.CreateUserRequest
		wantCode connect.Code
		wantErr  bool
	}{
		{
			name:   "success",
			svc:    okSvc,
			claims: adminClaims(),
			req:    &pb.CreateUserRequest{Email: "new@example.com", Name: "New", Password: "password123", Role: pb.Role_ROLE_MEMBER},
		},
		{
			name:     "access denied — non-admin",
			svc:      okSvc,
			claims:   memberClaims(),
			req:      &pb.CreateUserRequest{Email: "new@example.com", Name: "New", Password: "password123"},
			wantCode: connect.CodePermissionDenied,
			wantErr:  true,
		},
		{
			name:     "access denied — no claims",
			svc:      okSvc,
			claims:   nil,
			req:      &pb.CreateUserRequest{Email: "new@example.com", Name: "New", Password: "password123"},
			wantCode: connect.CodePermissionDenied,
			wantErr:  true,
		},
		{
			name:     "service error",
			svc:      &mockUserService{createFn: func(_ context.Context, _ user.User, _ string) (user.User, error) { return user.User{}, errors.New("db error") }},
			claims:   adminClaims(),
			req:      &pb.CreateUserRequest{Email: "fail@example.com", Name: "Fail", Password: "password123"},
			wantCode: connect.CodeInternal,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := newTestServer(t, tt.svc, tt.claims)
			_, err := client.CreateUser(context.Background(), connect.NewRequest(tt.req))
			if tt.wantErr {
				if err == nil {
					t.Fatal("CreateUser() expected error, got nil")
				}
				assertConnectCode(t, err, tt.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("CreateUser() unexpected error: %v", err)
			}
		})
	}
}

func TestHandler_ListUsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		svc      user.UserService
		claims   *authz.Claims
		wantCode connect.Code
		wantErr  bool
		wantLen  int
	}{
		{
			name: "success",
			svc: &mockUserService{
				listFn: func(_ context.Context) ([]user.User, error) {
					return []user.User{{ID: "1", Email: "a@example.com", Name: "A", Role: user.RoleMember}}, nil
				},
			},
			claims:  adminClaims(),
			wantLen: 1,
		},
		{
			name:     "access denied — non-admin",
			svc:      &mockUserService{listFn: func(_ context.Context) ([]user.User, error) { return nil, nil }},
			claims:   memberClaims(),
			wantCode: connect.CodePermissionDenied,
			wantErr:  true,
		},
		{
			name:     "access denied — no claims",
			svc:      &mockUserService{listFn: func(_ context.Context) ([]user.User, error) { return nil, nil }},
			claims:   nil,
			wantCode: connect.CodePermissionDenied,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := newTestServer(t, tt.svc, tt.claims)
			resp, err := client.ListUsers(context.Background(), connect.NewRequest(&pb.ListUsersRequest{}))
			if tt.wantErr {
				if err == nil {
					t.Fatal("ListUsers() expected error, got nil")
				}
				assertConnectCode(t, err, tt.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("ListUsers() unexpected error: %v", err)
			}
			if len(resp.Msg.GetUsers()) != tt.wantLen {
				t.Errorf("ListUsers() len = %d, want %d", len(resp.Msg.GetUsers()), tt.wantLen)
			}
		})
	}
}

func TestHandler_GetUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		svc      user.UserService
		claims   *authz.Claims
		id       string
		wantCode connect.Code
		wantErr  bool
	}{
		{
			name: "success",
			svc: &mockUserService{
				getFn: func(_ context.Context, id string) (user.User, error) {
					return user.User{ID: id, Email: "a@example.com", Name: "A", Role: user.RoleMember}, nil
				},
			},
			claims: adminClaims(),
			id:     "user-1",
		},
		{
			name:     "access denied — non-admin",
			svc:      &mockUserService{getFn: func(_ context.Context, _ string) (user.User, error) { return user.User{}, nil }},
			claims:   memberClaims(),
			id:       "user-1",
			wantCode: connect.CodePermissionDenied,
			wantErr:  true,
		},
		{
			name: "not found",
			svc: &mockUserService{
				getFn: func(_ context.Context, _ string) (user.User, error) {
					return user.User{}, user.ErrNotFound
				},
			},
			claims:   adminClaims(),
			id:       "missing",
			wantCode: connect.CodeNotFound,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := newTestServer(t, tt.svc, tt.claims)
			_, err := client.GetUser(context.Background(), connect.NewRequest(&pb.GetUserRequest{Id: tt.id}))
			if tt.wantErr {
				if err == nil {
					t.Fatal("GetUser() expected error, got nil")
				}
				assertConnectCode(t, err, tt.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("GetUser() unexpected error: %v", err)
			}
		})
	}
}

func TestHandler_UpdateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		svc      user.UserService
		claims   *authz.Claims
		req      *pb.UpdateUserRequest
		wantCode connect.Code
		wantErr  bool
	}{
		{
			name: "success",
			svc: &mockUserService{
				updateFn: func(_ context.Context, u user.User) (user.User, error) { return u, nil },
			},
			claims: adminClaims(),
			req:    &pb.UpdateUserRequest{Id: "user-1", Email: "updated@example.com", Name: "Updated", Role: pb.Role_ROLE_ADMIN},
		},
		{
			name:     "access denied — non-admin",
			svc:      &mockUserService{updateFn: func(_ context.Context, u user.User) (user.User, error) { return u, nil }},
			claims:   memberClaims(),
			req:      &pb.UpdateUserRequest{Id: "user-1"},
			wantCode: connect.CodePermissionDenied,
			wantErr:  true,
		},
		{
			name: "service error",
			svc: &mockUserService{
				updateFn: func(_ context.Context, _ user.User) (user.User, error) { return user.User{}, errors.New("db error") },
			},
			claims:   adminClaims(),
			req:      &pb.UpdateUserRequest{Id: "user-1", Email: "fail@example.com", Name: "Fail", Role: pb.Role_ROLE_MEMBER},
			wantCode: connect.CodeInternal,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := newTestServer(t, tt.svc, tt.claims)
			_, err := client.UpdateUser(context.Background(), connect.NewRequest(tt.req))
			if tt.wantErr {
				if err == nil {
					t.Fatal("UpdateUser() expected error, got nil")
				}
				assertConnectCode(t, err, tt.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("UpdateUser() unexpected error: %v", err)
			}
		})
	}
}

func TestHandler_DeleteUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		svc      user.UserService
		claims   *authz.Claims
		id       string
		wantCode connect.Code
		wantErr  bool
	}{
		{
			name:   "success",
			svc:    &mockUserService{deleteFn: func(_ context.Context, _ string) error { return nil }},
			claims: adminClaims(),
			id:     "user-1",
		},
		{
			name:     "access denied — non-admin",
			svc:      &mockUserService{deleteFn: func(_ context.Context, _ string) error { return nil }},
			claims:   memberClaims(),
			id:       "user-1",
			wantCode: connect.CodePermissionDenied,
			wantErr:  true,
		},
		{
			name:     "access denied — no claims",
			svc:      &mockUserService{deleteFn: func(_ context.Context, _ string) error { return nil }},
			claims:   nil,
			id:       "user-1",
			wantCode: connect.CodePermissionDenied,
			wantErr:  true,
		},
		{
			name:     "service error",
			svc:      &mockUserService{deleteFn: func(_ context.Context, _ string) error { return errors.New("db error") }},
			claims:   adminClaims(),
			id:       "missing",
			wantCode: connect.CodeInternal,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := newTestServer(t, tt.svc, tt.claims)
			_, err := client.DeleteUser(context.Background(), connect.NewRequest(&pb.DeleteUserRequest{Id: tt.id}))
			if tt.wantErr {
				if err == nil {
					t.Fatal("DeleteUser() expected error, got nil")
				}
				assertConnectCode(t, err, tt.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("DeleteUser() unexpected error: %v", err)
			}
		})
	}
}
