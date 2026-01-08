.PHONY: run build test test-race mock clean help

# Default target
.DEFAULT_GOAL := help

# Run the application
run:
	@echo "Running application..."
	go run ./cmd/server

# Build the application
build:
	@echo "Building application..."
	@mkdir -p bin
	go build -o bin/server ./cmd/server
	@echo "Build complete: bin/server"

# Run tests
test:
	@echo "Running tests..."
	go test ./... -v

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race ./...

# Generate mocks using mockery
mock:
	@echo "Generating mocks..."
	@if command -v mockery >/dev/null 2>&1; then \
		mockery; \
	else \
		echo "Error: mockery not found. Install with: go install github.com/vektra/mockery/v2@latest"; \
		exit 1; \
	fi

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test ./... -coverprofile=coverage.txt -covermode=atomic
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf mocks/
	rm -f coverage.txt coverage.html
	@echo "Clean complete"

# Show help
help:
	@echo "Available targets:"
	@echo "  run         - Run the application"
	@echo "  build       - Build the application binary"
	@echo "  test        - Run tests"
	@echo "  test-race   - Run tests with race detector"
	@echo "  mock        - Generate mocks using mockery"
	@echo "  coverage    - Run tests with coverage report"
	@echo "  tidy        - Tidy dependencies"
	@echo "  clean       - Clean build artifacts"
	@echo "  help        - Show this help message"
