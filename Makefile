.PHONY: build run test clean docker-up docker-down migrate-up migrate-down tidy

# Build the binary
build:
	go build -o bin/spotifish ./cmd/server

# Run locally (requires .env file and running Postgres)
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Docker
docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

# Tidy Go modules
tidy:
	go mod tidy

# Lint (requires golangci-lint)
lint:
	golangci-lint run ./...
