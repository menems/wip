package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/menems/sass/pkg/storage/postgres"
)

func TestNew(t *testing.T) {
	t.Parallel()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()

	pool, err := postgres.New(ctx, dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(pool.Close)

	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
}

func TestNew_InvalidDSN(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	_, err := postgres.New(ctx, "postgres://invalid-host:5432/db")
	if err == nil {
		t.Fatal("expected error for unreachable host, got nil")
	}
}
