.PHONY: build run dev docker-up docker-down docker-build docker-prod clean test fmt lint

# Build the Go server binary
build:
	go build -o bin/server ./cmd/server

# Run the server locally
run: build
	./bin/server

# Run with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air -c .air.toml

# Start dev containers (PostgreSQL + PgAdmin only)
docker-up:
	docker compose up -d postgres pgadmin

# Stop all containers
docker-down:
	docker compose down

# Build production Docker image
docker-build:
	docker compose build app

# Start full production stack (DB + App + PgAdmin)
docker-prod:
	docker compose up -d

# Remove volumes (WARNING: deletes all data)
docker-clean:
	docker compose down -v

# Build frontend
web-build:
	cd web && npm run build

# Run tests
test:
	go test ./... -v

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Lint
lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
	rm -rf web/dist/
