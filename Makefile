# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=gorm-migrate
BINARY_PATH=bin/$(BINARY_NAME)

# Directories
BIN_DIR=bin
MIGRATIONS_DIR=migrations

.PHONY: all build clean test deps migrate-create migrate-up migrate-down lint security-check

all: clean build

# Build the migration tool
build:
	@echo "Building gorm-migrate tool..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/gorm-schema

# Clean build files
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	$(GOCLEAN)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./tests/...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) tidy

# Create a new migration
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: Migration name is required. Usage: make migrate-create name=migration_name"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(name)..."
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/gorm-schema
	./$(BINARY_PATH) create $(name)

# Apply pending migrations
migrate-up:
	@echo "Applying pending migrations..."
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/gorm-schema
	./$(BINARY_PATH) up

# Rollback the last migration
migrate-down:
	@echo "Rolling back last migration..."
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/gorm-schema
	./$(BINARY_PATH) down

# Run linters
lint:
	@echo "Running linters..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2; \
	fi
	golangci-lint run --enable=govet,staticcheck --disable=errcheck,ineffassign,unused --timeout=5m ./...


# Show help
help:
	@echo "Available commands:"
	@echo "  make build          - Build the gorm-migrate tool"
	@echo "  make clean          - Clean build files"
	@echo "  make test           - Run tests"
	@echo "  make deps           - Install dependencies"
	@echo "  make lint           - Run linters"
	@echo "  make migrate-create - Create a new migration (requires name=migration_name)"
	@echo "  make migrate-up     - Apply pending migrations"
	@echo "  make migrate-down   - Rollback the last migration"
	@echo "  make help           - Show this help message" 