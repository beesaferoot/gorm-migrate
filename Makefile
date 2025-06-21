# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
MIGRATE_BINARY=bin/migrate

# Directories
BIN_DIR=bin
MIGRATIONS_DIR=migrations

.PHONY: all build clean test deps migrate-create migrate-up migrate-down lint security-check

all: clean build

# Build the migration tool
build:
	@echo "Building migration tool..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(MIGRATE_BINARY) ./cmd/migrate

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
	$(GOBUILD) -o $(MIGRATE_BINARY) ./cmd/migrate
	./$(MIGRATE_BINARY) -action create -name $(name)

# Apply pending migrations
migrate-up:
	@echo "Applying pending migrations..."
	$(GOBUILD) -o $(MIGRATE_BINARY) ./cmd/migrate
	./$(MIGRATE_BINARY) -action up

# Rollback the last migration
migrate-down:
	@echo "Rolling back last migration..."
	$(GOBUILD) -o $(MIGRATE_BINARY) ./cmd/migrate
	./$(MIGRATE_BINARY) -action down

# Run linters
lint:
	@echo "Running linters..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2; \
	fi
	golangci-lint run ./...

# Run security checks
security-check:
	@echo "Running security checks..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	gosec ./...

# Show help
help:
	@echo "Available commands:"
	@echo "  make build          - Build the migration tool"
	@echo "  make clean          - Clean build files"
	@echo "  make test           - Run tests"
	@echo "  make deps           - Install dependencies"
	@echo "  make lint           - Run linters"
	@echo "  make security-check - Run security checks"
	@echo "  make migrate-create - Create a new migration (requires name=migration_name)"
	@echo "  make migrate-up     - Apply pending migrations"
	@echo "  make migrate-down   - Rollback the last migration"
	@echo "  make help           - Show this help message" 