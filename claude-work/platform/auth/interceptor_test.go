package auth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/blaz/serve/platform/auth"
)

func TestWithUserID_RoundTrip(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000000")
	ctx := auth.WithUserID(context.Background(), id)

	got, ok := auth.UserIDFromContext(ctx)
	if !ok {
		t.Fatal("expected userID in context")
	}
	if got != id {
		t.Fatalf("want %v, got %v", id, got)
	}
}

func TestUserIDFromContext_Empty(t *testing.T) {
	_, ok := auth.UserIDFromContext(context.Background())
	if ok {
		t.Fatal("expected no userID in empty context")
	}
}
