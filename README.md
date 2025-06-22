# GORM Schema Migration Tool

[![CI](https://github.com/beesaferoot/gorm-schema/workflows/CI/badge.svg)](https://github.com/beesaferoot/gorm-schema/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/beesaferoot/gorm-schema)](https://goreportcard.com/report/github.com/beesaferoot/gorm-schema)
[![Go Version](https://img.shields.io/github/go-mod/go-version/beesaferoot/gorm-schema)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Automatically generate database migrations from your GORM models by comparing them with the current database schema.

## Features

- 🔄 **Auto-generate migrations** from GORM model changes
- 🎯 **Smart comparison** - only generates migrations for actual changes
- 📊 **Case-insensitive** table and column name handling
- 🚀 **Zero false positives** with advanced type normalization

## Quick Start

### 1. Install

```bash
go install github.com/beesaferoot/gorm-schema/cmd/gorm-schema@latest
```

### 2. Set up environment

```bash
export DATABASE_URL="postgresql://user:pass@localhost:5432/your_db"
```

### 3. Create your migration binary

Create `cmd/migration/main.go` in your project:

```go
package main

import (
    "reflect"
    "your-project/models" // Import your models package
    "github.com/spf13/cobra"
    "github.com/beesaferoot/gorm-schema/internal/migration"
    "github.com/beesaferoot/gorm-schema/internal/migration/commands"
)

// Simple registry implementation
type MyModelRegistry struct{}

func (r *MyModelRegistry) GetModelTypes() map[string]reflect.Type {
    return models.ModelTypeRegistry // Your registry
}

func init() {
    migration.GlobalModelRegistry = &MyModelRegistry{}
}

func main() {
    rootCmd := &cobra.Command{
        Use:   "migration",
        Short: "Database Migration Tool",
    }

    rootCmd.AddCommand(
        commands.InitCmd(),
        commands.CreateCmd(),
        commands.GenerateCmd(),
        commands.UpCmd(),
        commands.DownCmd(),
        commands.StatusCmd(),
        commands.HistoryCmd(),
        commands.ValidateCmd(),
    )

    if err := rootCmd.Execute(); err != nil {
        panic(err)
    }
}
```

### 4. Create your model registry

Create `models/models_registry.go` in your project:

```go
package models

import "reflect"

var ModelTypeRegistry = map[string]reflect.Type{
    "User":    reflect.TypeOf(User{}),
    "Post":    reflect.TypeOf(Post{}),
    "Comment": reflect.TypeOf(Comment{}),
    // Add all your models here
}
```

### 5. Initialize and generate migrations

```bash
go run cmd/migration/main.go init
go run cmd/migration/main.go generate init_db
go run cmd/migration/main.go up
```

## Usage

```bash
# Initialize migration system
go run cmd/migration/main.go init

# Generate migration from model changes
go run cmd/migration/main.go generate <name>

# Apply migrations
go run cmd/migration/main.go up

# Rollback last migration
go run cmd/migration/main.go down

# Check status
go run cmd/migration/main.go status
```

## Example

Your GORM model:

```go
type User struct {
    gorm.Model
    Name  string
    Email string `gorm:"uniqueIndex"`
}
```

Generated migration:

```go
package migrations

import (
    "github.com/beesaferoot/gorm-schema/internal/migration"
    "gorm.io/gorm"
    "time"
)

func init() {
    migration.RegisterMigration(&migration.Migration{
        Version:   "20250620232804",
        Name:      "init_db",
        CreatedAt: time.Now(),
        Up: func(db *gorm.DB) error {
            if err := db.Exec(`CREATE TABLE "users" (
    id BIGSERIAL PRIMARY KEY,
    created_at timestamp,
    updated_at timestamp,
    deleted_at timestamp,
    name varchar(255),
    email varchar(255)
);`).Error; err != nil {
                return err
            }
            if err := db.Exec(`CREATE UNIQUE INDEX idx_users_email ON "users"(email);`).Error; err != nil {
                return err
            }
            return nil
        },
        Down: func(db *gorm.DB) error {
            if err := db.Exec(`DROP TABLE IF EXISTS "users";`).Error; err != nil {
                return err
            }
            return nil
        },
    })
}
```

## Environment Variables

| Variable          | Description                  | Required                     |
| ----------------- | ---------------------------- | ---------------------------- |
| `DATABASE_URL`    | PostgreSQL connection string | Yes                          |
| `MIGRATIONS_PATH` | Path for migration files     | No (default: `./migrations`) |

## Testing

### Run All Tests

```bash
make test
```

### Run Tests with Coverage

```bash
go test -v -coverprofile=coverage.out ./tests/...
go tool cover -func=coverage.out
```

### Run Specific Test Packages

```bash
# Run diff tests
go test ./tests/migration/diff/...

# Run command tests
go test ./tests/migration/...

# Run with verbose output
go test -v ./tests/...
```

### Run Linters

```bash
make lint
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
