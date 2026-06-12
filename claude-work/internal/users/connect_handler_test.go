package users_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	usersv1 "github.com/blaz/serve/gen/users/v1"
	"github.com/blaz/serve/gen/users/v1/usersv1connect"
	"github.com/blaz/serve/internal/users"
)

func newConnectServer() *httptest.Server {
	repo := users.NewRepository()
	svc := users.NewService(repo)
	h := users.NewConnectHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

func connectClient(srv *httptest.Server) usersv1connect.UserServiceClient {
	return usersv1connect.NewUserServiceClient(http.DefaultClient, srv.URL)
}

func TestConnectHandler_Register(t *testing.T) {
	srv := newConnectServer()
	defer srv.Close()
	c := connectClient(srv)
	ctx := context.Background()

	resp, err := c.Register(ctx, connect.NewRequest(&usersv1.RegisterRequest{
		Name: "Alice", Email: "alice@example.com", Password: "secret",
	}))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.Msg.Name != "Alice" || resp.Msg.Email != "alice@example.com" || resp.Msg.Id == "" {
		t.Fatalf("unexpected response %+v", resp.Msg)
	}
}

func TestConnectHandler_Register_Conflict(t *testing.T) {
	srv := newConnectServer()
	defer srv.Close()
	c := connectClient(srv)
	ctx := context.Background()

	c.Register(ctx, connect.NewRequest(&usersv1.RegisterRequest{ //nolint
		Name: "Alice", Email: "alice@example.com", Password: "secret",
	}))
	_, err := c.Register(ctx, connect.NewRequest(&usersv1.RegisterRequest{
		Name: "Other", Email: "alice@example.com", Password: "other",
	}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("want AlreadyExists, got %v", connect.CodeOf(err))
	}
}

func TestConnectHandler_Register_MissingFields(t *testing.T) {
	srv := newConnectServer()
	defer srv.Close()
	c := connectClient(srv)
	ctx := context.Background()

	_, err := c.Register(ctx, connect.NewRequest(&usersv1.RegisterRequest{Name: "Alice"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestConnectHandler_SignIn(t *testing.T) {
	srv := newConnectServer()
	defer srv.Close()
	c := connectClient(srv)
	ctx := context.Background()

	c.Register(ctx, connect.NewRequest(&usersv1.RegisterRequest{ //nolint
		Name: "Bob", Email: "bob@example.com", Password: "pass",
	}))

	resp, err := c.SignIn(ctx, connect.NewRequest(&usersv1.SignInRequest{
		Email: "bob@example.com", Password: "pass",
	}))
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	if resp.Msg.Token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestConnectHandler_SignIn_BadCredentials(t *testing.T) {
	srv := newConnectServer()
	defer srv.Close()
	c := connectClient(srv)
	ctx := context.Background()

	c.Register(ctx, connect.NewRequest(&usersv1.RegisterRequest{ //nolint
		Name: "Bob", Email: "bob@example.com", Password: "pass",
	}))

	_, err := c.SignIn(ctx, connect.NewRequest(&usersv1.SignInRequest{
		Email: "bob@example.com", Password: "wrong",
	}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("want Unauthenticated, got %v", connect.CodeOf(err))
	}
}

func TestConnectHandler_SignIn_NotFound(t *testing.T) {
	srv := newConnectServer()
	defer srv.Close()
	c := connectClient(srv)

	_, err := c.SignIn(context.Background(), connect.NewRequest(&usersv1.SignInRequest{
		Email: "ghost@example.com", Password: "pass",
	}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("want Unauthenticated, got %v", connect.CodeOf(err))
	}
}
