.PHONY: build run test lint vet tidy

build:
	go build ./cmd/serve/

run:
	go run ./cmd/serve/

test:
	go test ./...

lint:
	go vet ./...

vet: lint

tidy:
	go mod tidy
