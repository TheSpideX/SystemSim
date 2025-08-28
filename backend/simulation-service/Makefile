# Simulation Service Makefile

.PHONY: help build run test clean docker-build docker-run docker-stop deps

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the simulation service binary"
	@echo "  run          - Run the simulation service locally"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  docker-stop  - Stop Docker Compose services"
	@echo "  docker-logs  - View Docker Compose logs"

# Build the binary
build:
	@echo "Building simulation service..."
	go build -o bin/simulation-service cmd/server/main.go

# Run the service locally
run:
	@echo "Starting simulation service..."
	go run cmd/server/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

# Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t simulation-service:latest .

docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-stop:
	@echo "Stopping Docker Compose services..."
	docker-compose down

docker-logs:
	@echo "Viewing Docker Compose logs..."
	docker-compose logs -f

# Development helpers
dev-setup: deps
	@echo "Setting up development environment..."
	@echo "Creating bin directory..."
	mkdir -p bin

# Check if service is running
check:
	@echo "Checking service health..."
	curl -f http://localhost:11000/health || echo "Service not running"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# Security scan (requires gosec)
security:
	@echo "Running security scan..."
	gosec ./...
