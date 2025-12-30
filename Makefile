.PHONY: build run test clean migrate-up migrate-down docker-build docker-run

# Build the application
build:
	go build -o bin/server ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run database migrations up
migrate-up:
	migrate -path migrations -database "sqlite3://vocabulator.db" up

# Run database migrations down
migrate-down:
	migrate -path migrations -database "sqlite3://vocabulator.db" down

# Build Docker image
docker-build:
	docker build -t vocabulator .

# Run with Docker Compose
docker-run:
	docker-compose up --build

# Stop Docker Compose
docker-stop:
	docker-compose down

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run
