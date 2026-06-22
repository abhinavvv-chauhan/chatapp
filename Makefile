.PHONY: run build test up down

# Run the Go application
run:
	go run cmd/server/main.go

# Build the Go binary
build:
	go build -o bin/server cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Start Docker containers (Postgres, Redis)
up:
	docker-compose up -d

# Stop Docker containers
down:
	docker-compose down