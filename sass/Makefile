.PHONY: build run test coverage vet lint clean generate sqlc-generate migrate-up migrate-down \
        db-up db-down db-reset db-logs

BIN := server
PORT ?= 1664
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/sass?sslmode=disable

build:
	go build -o $(BIN) ./cmd/api

run:
	PORT=$(PORT) DATABASE_URL=$(DATABASE_URL) go run main.go

test:
	go test -race -count=1 ./...

coverage:
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vet:
	go vet ./...

lint: vet
	golangci-lint run

clean:
	rm -f $(BIN) coverage.out

generate: sqlc-generate

sqlc-generate:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.26.0 generate

migrate-up:
	go run -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1 -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	go run -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1 -path db/migrations -database "$(DATABASE_URL)" down

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-reset:
	docker compose down -v
	docker compose up -d postgres

db-logs:
	docker compose logs -f postgres
