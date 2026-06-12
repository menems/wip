package contacts_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/blaz/serve/internal/contacts"
)

var (
	userA = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000000")
	userB = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000000")
)

func newTestService() contacts.Service {
	return contacts.NewService(contacts.NewRepository())
}

func TestService_AddAndGetByName(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	c := contacts.Contact{Name: "Alice", Phone: "555-1234", Email: "alice@example.com"}
	if err := svc.Add(ctx, userA, c); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := svc.GetByName(ctx, userA, "Alice")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	c.UserID = userA
	if got != c {
		t.Fatalf("want %+v, got %+v", c, got)
	}
}

func TestService_AddAndGetByName_WithAddress(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	c := contacts.Contact{
		Name:  "Bob",
		Phone: "555-9999",
		Email: "bob@example.com",
		Address: contacts.Address{
			Street:  "123 Main St",
			City:    "Springfield",
			State:   "IL",
			Zip:     "62701",
			Country: "US",
		},
	}
	if err := svc.Add(ctx, userA, c); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := svc.GetByName(ctx, userA, "Bob")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	c.UserID = userA
	if got != c {
		t.Fatalf("want %+v, got %+v", c, got)
	}
}

func TestService_AddAndGetByName_NoAddress(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	c := contacts.Contact{Name: "Carol", Phone: "555-0000", Email: "carol@example.com"}
	if err := svc.Add(ctx, userA, c); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := svc.GetByName(ctx, userA, "Carol")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.Address != (contacts.Address{}) {
		t.Fatalf("want zero address, got %+v", got.Address)
	}
}

func TestService_AddConflict(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	c := contacts.Contact{Name: "Alice", Phone: "555-1234", Email: "alice@example.com"}
	_ = svc.Add(ctx, userA, c)
	err := svc.Add(ctx, userA, c)
	if !errors.Is(err, contacts.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestService_List(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_ = svc.Add(ctx, userA, contacts.Contact{Name: "Alice", Phone: "1", Email: "a@x.com"})
	_ = svc.Add(ctx, userA, contacts.Contact{Name: "Bob", Phone: "2", Email: "b@x.com"})

	list, err := svc.List(ctx, userA)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 contacts, got %d", len(list))
	}
}

func TestService_DeleteThenGet(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	c := contacts.Contact{Name: "Alice", Phone: "555-1234", Email: "alice@example.com"}
	_ = svc.Add(ctx, userA, c)

	if err := svc.Delete(ctx, userA, "Alice"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := svc.GetByName(ctx, userA, "Alice")
	if !errors.Is(err, contacts.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestService_GetByName_NotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.GetByName(context.Background(), userA, "Ghost")
	if !errors.Is(err, contacts.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := newTestService()
	err := svc.Delete(context.Background(), userA, "Ghost")
	if !errors.Is(err, contacts.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestService_UserIsolation(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	c := contacts.Contact{Name: "Alice", Phone: "555-1234", Email: "alice@example.com"}
	_ = svc.Add(ctx, userA, c)

	t.Run("other user cannot get", func(t *testing.T) {
		_, err := svc.GetByName(ctx, userB, "Alice")
		if !errors.Is(err, contacts.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("other user list is empty", func(t *testing.T) {
		list, err := svc.List(ctx, userB)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("want 0 contacts, got %d", len(list))
		}
	})

	t.Run("other user cannot delete", func(t *testing.T) {
		err := svc.Delete(ctx, userB, "Alice")
		if !errors.Is(err, contacts.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("same name allowed for different users", func(t *testing.T) {
		if err := svc.Add(ctx, userB, c); err != nil {
			t.Fatalf("Add for userB: %v", err)
		}
	})
}
