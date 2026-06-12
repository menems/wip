.PHONY: build run migrate-up migrate-down migrate-version migrate-force sqlc

MIGRATIONS_PATH := internal/db/migrations

# ── Build ──────────────────────────────────────────────────────────────────────

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

# ── Migrations (requires: go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest)

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$$DATABASE_URL" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$$DATABASE_URL" down 1

migrate-version:
	migrate -path $(MIGRATIONS_PATH) -database "$$DATABASE_URL" version

## Usage: make migrate-force N=2
migrate-force:
	migrate -path $(MIGRATIONS_PATH) -database "$$DATABASE_URL" force $(N)

# ── Code generation ────────────────────────────────────────────────────────────

sqlc:
	sqlc generate
