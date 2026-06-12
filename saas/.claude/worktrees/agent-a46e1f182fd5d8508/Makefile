.DEFAULT_GOAL := help

.PHONY: generate lint test migrate-up migrate-down migrate-create \
        docker-up docker-down run dev front help

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/saas?sslmode=disable
JWT_SECRET   ?= dev-secret-change-me
ADDR         ?= :8080

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

generate: ## Run buf and sqlc code generation
	go tool buf generate
	go tool sqlc generate

lint: ## Run buf lint and go vet
	go tool buf lint
	go vet ./...

test: ## Run tests with race detector
	go test -race -count=1 ./...

docker-up: ## Start Postgres via Docker Compose (waits until healthy)
	docker compose up -d --wait

docker-down: ## Stop and remove Docker Compose containers
	docker compose down

migrate-up: ## Run all up migrations (requires DB to be running)
	go run -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1 -path db/migrations -database "$(DATABASE_URL)" up

migrate-down: ## Roll back the last migration
	go run -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1 -path db/migrations -database "$(DATABASE_URL)" down 1

migrate-create: ## Create a new migration (usage: make migrate-create NAME=add_foo)
	go tool migrate create -ext sql -dir db/migrations -seq $(NAME)

run: ## Start the server (requires DB running and migrations applied)
	ADDR=$(ADDR) JWT_SECRET=$(JWT_SECRET) DATABASE_URL=$(DATABASE_URL) \
		go run cmd/server/main.go

front: ## Start the frontend dev server (www/)
	cd www && npm run dev

dev: docker-up migrate-up run ## Full local dev start: DB + migrations + server
