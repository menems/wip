// Package db provides the PostgreSQL connection pool and migration runner.
package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates and validates a pgx connection pool using the given DSN.
// Returns an error if the pool cannot be created or the database is unreachable.
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("db: create pool: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	return pool, nil
}

// Migrate runs all *.sql files found in migrationsDir in lexicographic order.
// Files that have already been applied are tracked in a migrations table.
// This is a simple numbered-file runner — no down migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	// Ensure the migrations tracking table exists.
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return fmt.Errorf("db: migrate: ensure table: %w", err)
	}

	// List and sort migration files.
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("db: migrate: read dir %q: %w", migrationsDir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		applied, err := isMigrationApplied(ctx, pool, name)
		if err != nil {
			return fmt.Errorf("db: migrate: check %q: %w", name, err)
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return fmt.Errorf("db: migrate: read %q: %w", name, err)
		}

		if err = applyMigration(ctx, pool, name, string(content)); err != nil {
			return fmt.Errorf("db: migrate: apply %q: %w", name, err)
		}
	}

	return nil
}

// Ping checks that the database is reachable.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("db: ping: %w", err)
	}
	return nil
}

// ensureMigrationsTable creates the schema_migrations tracking table if it does not exist.
func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	return err
}

// isMigrationApplied reports whether the given migration filename is recorded.
func isMigrationApplied(ctx context.Context, pool *pgxpool.Pool, filename string) (bool, error) {
	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM schema_migrations WHERE filename = $1`, filename,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// applyMigration executes the SQL and records the filename in a single transaction.
func applyMigration(ctx context.Context, pool *pgxpool.Pool, filename, sql string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err = tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if _, err = tx.Exec(ctx,
		`INSERT INTO schema_migrations (filename) VALUES ($1)`, filename,
	); err != nil {
		return fmt.Errorf("record: %w", err)
	}

	return tx.Commit(ctx)
}
