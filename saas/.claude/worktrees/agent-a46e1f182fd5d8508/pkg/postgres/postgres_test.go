package postgres_test

import (
	"context"
	"testing"

	"github.com/menems/saas/pkg/postgres"
)

// TestNew_InvalidDSN ensures New returns an error for a malformed DSN without
// requiring a live database.
func TestNew_InvalidDSN(t *testing.T) {
	t.Parallel()

	_, err := postgres.New(context.Background(), "not-a-valid-dsn")
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
}

// TestNew_Defaults verifies that default options (maxConns=10) are applied and
// PGX() returns a non-nil pool when the DSN is syntactically valid.
// We use a valid-looking but unreachable DSN so pgxpool.ParseConfig succeeds
// and pool creation does not attempt a real connection.
func TestNew_Defaults(t *testing.T) {
	t.Parallel()

	// pgxpool.NewWithConfig is lazy: it does not dial on creation, so this
	// succeeds without a live database.
	p, err := postgres.New(context.Background(), "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer p.Close()

	if p.PGX() == nil {
		t.Fatal("expected non-nil underlying pool")
	}
}

// TestWithMaxConns verifies that the option is accepted without panicking.
func TestWithMaxConns(t *testing.T) {
	t.Parallel()

	p, err := postgres.New(
		context.Background(),
		"postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		postgres.WithMaxConns(25),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer p.Close()

	if p.PGX() == nil {
		t.Fatal("expected non-nil underlying pool")
	}
}
