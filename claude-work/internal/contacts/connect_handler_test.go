package contacts_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	addressbookv1 "github.com/blaz/serve/gen/addressbook/v1"
	"github.com/blaz/serve/gen/addressbook/v1/addressbookv1connect"
	"github.com/blaz/serve/internal/contacts"
	"github.com/blaz/serve/platform/auth"
)

// fixedUserInterceptor is a test-only interceptor that injects a hardcoded userID.
func fixedUserInterceptor(id uuid.UUID) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return next(auth.WithUserID(ctx, id), req)
		}
	})
}

func newConnectServer(userID uuid.UUID) *httptest.Server {
	repo := contacts.NewRepository()
	svc := contacts.NewService(repo)
	h := contacts.NewConnectHandler(svc, fixedUserInterceptor(userID))
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

// newConnectServerNoAuth sets up a server with no auth interceptor (unauthenticated).
func newConnectServerNoAuth() *httptest.Server {
	repo := contacts.NewRepository()
	svc := contacts.NewService(repo)
	h := contacts.NewConnectHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

func connectClient(srv *httptest.Server) addressbookv1connect.ContactServiceClient {
	return addressbookv1connect.NewContactServiceClient(http.DefaultClient, srv.URL)
}

func TestConnectHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("add and get with address", func(t *testing.T) {
		srv := newConnectServer(userA)
		defer srv.Close()
		c := connectClient(srv)

		addr := &addressbookv1.Address{
			Street: "123 Main St", City: "Springfield",
			State: "IL", Zip: "62701", Country: "US",
		}
		_, err := c.Add(ctx, connect.NewRequest(&addressbookv1.AddRequest{
			Name: "Dave", Phone: "555", Email: "d@b.com", Address: addr,
		}))
		if err != nil {
			t.Fatal(err)
		}

		resp, err := c.GetByName(ctx, connect.NewRequest(&addressbookv1.GetByNameRequest{Name: "Dave"}))
		if err != nil {
			t.Fatal(err)
		}
		got := resp.Msg.Contact.Address
		if got == nil || got.Street != addr.Street || got.City != addr.City || got.Zip != addr.Zip {
			t.Errorf("got address %+v, want %+v", got, addr)
		}
	})

	t.Run("add and get", func(t *testing.T) {
		srv := newConnectServer(userA)
		defer srv.Close()
		c := connectClient(srv)

		_, err := c.Add(ctx, connect.NewRequest(&addressbookv1.AddRequest{
			Name: "Alice", Phone: "123", Email: "a@b.com",
		}))
		if err != nil {
			t.Fatal(err)
		}

		resp, err := c.GetByName(ctx, connect.NewRequest(&addressbookv1.GetByNameRequest{Name: "Alice"}))
		if err != nil {
			t.Fatal(err)
		}
		if resp.Msg.Contact.Email != "a@b.com" {
			t.Errorf("got email %q, want %q", resp.Msg.Contact.Email, "a@b.com")
		}
	})

	t.Run("list", func(t *testing.T) {
		srv := newConnectServer(userA)
		defer srv.Close()
		c := connectClient(srv)

		for _, name := range []string{"Alice", "Bob"} {
			c.Add(ctx, connect.NewRequest(&addressbookv1.AddRequest{ //nolint
				Name: name, Phone: "000", Email: name + "@x.com",
			}))
		}

		resp, err := c.List(ctx, connect.NewRequest(&addressbookv1.ListRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Msg.Contacts) != 2 {
			t.Errorf("got %d contacts, want 2", len(resp.Msg.Contacts))
		}
	})

	t.Run("delete", func(t *testing.T) {
		srv := newConnectServer(userA)
		defer srv.Close()
		c := connectClient(srv)

		c.Add(ctx, connect.NewRequest(&addressbookv1.AddRequest{ //nolint
			Name: "Alice", Phone: "000", Email: "a@b.com",
		}))

		_, err := c.Delete(ctx, connect.NewRequest(&addressbookv1.DeleteRequest{Name: "Alice"}))
		if err != nil {
			t.Fatal(err)
		}

		_, err = c.GetByName(ctx, connect.NewRequest(&addressbookv1.GetByNameRequest{Name: "Alice"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("got code %v, want NotFound", connect.CodeOf(err))
		}
	})

	t.Run("add conflict", func(t *testing.T) {
		srv := newConnectServer(userA)
		defer srv.Close()
		c := connectClient(srv)

		c.Add(ctx, connect.NewRequest(&addressbookv1.AddRequest{ //nolint
			Name: "Alice", Phone: "000", Email: "a@b.com",
		}))
		_, err := c.Add(ctx, connect.NewRequest(&addressbookv1.AddRequest{
			Name: "Alice", Phone: "000", Email: "a@b.com",
		}))
		if connect.CodeOf(err) != connect.CodeAlreadyExists {
			t.Errorf("got code %v, want AlreadyExists", connect.CodeOf(err))
		}
	})

	t.Run("get not found", func(t *testing.T) {
		srv := newConnectServer(userA)
		defer srv.Close()
		c := connectClient(srv)

		_, err := c.GetByName(ctx, connect.NewRequest(&addressbookv1.GetByNameRequest{Name: "Ghost"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("got code %v, want NotFound", connect.CodeOf(err))
		}
	})

	t.Run("user isolation", func(t *testing.T) {
		// Two servers sharing the same repo, each injecting a different userID.
		repo := contacts.NewRepository()
		svcA := contacts.NewService(repo)
		svcB := contacts.NewService(repo)

		muxA := http.NewServeMux()
		contacts.NewConnectHandler(svcA, fixedUserInterceptor(userA)).RegisterRoutes(muxA)
		srvA := httptest.NewServer(muxA)
		defer srvA.Close()

		muxB := http.NewServeMux()
		contacts.NewConnectHandler(svcB, fixedUserInterceptor(userB)).RegisterRoutes(muxB)
		srvB := httptest.NewServer(muxB)
		defer srvB.Close()

		cA := addressbookv1connect.NewContactServiceClient(http.DefaultClient, srvA.URL)
		cB := addressbookv1connect.NewContactServiceClient(http.DefaultClient, srvB.URL)

		cA.Add(ctx, connect.NewRequest(&addressbookv1.AddRequest{ //nolint
			Name: "Alice", Phone: "000", Email: "a@b.com",
		}))

		_, err := cB.GetByName(ctx, connect.NewRequest(&addressbookv1.GetByNameRequest{Name: "Alice"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("got code %v, want NotFound", connect.CodeOf(err))
		}

		resp, err := cB.List(ctx, connect.NewRequest(&addressbookv1.ListRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Msg.Contacts) != 0 {
			t.Errorf("got %d contacts for userB, want 0", len(resp.Msg.Contacts))
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		srv := newConnectServerNoAuth()
		defer srv.Close()
		c := connectClient(srv)

		_, err := c.List(ctx, connect.NewRequest(&addressbookv1.ListRequest{}))
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Errorf("got code %v, want Unauthenticated", connect.CodeOf(err))
		}
	})
}
