.PHONY: dev build migrate test seed lint help

# Default target
.DEFAULT_GOAL := help

## dev: Start the full stack with hot reload (postgres + backend/air + frontend/Vite HMR)
dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

## build: Build all Docker images
build:
	docker compose build

## migrate: Run pending SQL migrations against DATABASE_URL
migrate:
	@echo "Running migrations..."
	@if [ -z "$$DATABASE_URL" ]; then \
		export DATABASE_URL=$$(grep DATABASE_URL .env | cut -d= -f2-); \
	fi; \
	docker compose run --rm backend migrate

## seed: Insert default admin role + admin user (idempotent)
seed:
	@echo "Seeding default admin role and user..."
	docker compose run --rm backend seed

## test: Run all Go unit + integration tests
test:
	@echo "Running backend tests..."
	cd backend && go test ./... -v -race -count=1

## lint: Run Go linter
lint:
	cd backend && go vet ./...

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /'
