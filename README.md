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
- 🚀 **Almost Zero false positives** with advanced type normalization

## Quick Start

### 1. Install

```bash
go get -u github.com/beesaferoot/gorm-schema@latest
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
    "your-project/models" // Import your models package
    "github.com/spf13/cobra"
    "github.com/joho/godotenv"
    "github.com/beesaferoot/gorm-schema/migration"
    "github.com/beesaferoot/gorm-schema/migration/commands"
)

// Simple registry implementation
type MyModelRegistry struct{}

func (r *MyModelRegistry) GetModels() map[string]interface{} {
    return models.ModelTypeRegistry // Your registry
}

func init() {
    migration.GlobalModelRegistry = &MyModelRegistry{}
}

func main() {
    _ = godotenv.Load() // optionally load environment file
    rootCmd := &cobra.Command{
        Use:   "migration",
        Short: "Database Migration Tool",
    }

    rootCmd.AddCommand(
        commands.RegisterCmd(),
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


### 4. Generate your model registry

Use the register command to automatically scan your models directory (e.g., models/) and generate a models_registry.go file.


```bash
go run cmd/migration/main.go register [path/to/models]
```

This command creates a standard Go file that you can review and even edit if needed. It will look something like this:

```go
package models

var ModelTypeRegistry = map[string]interface{}{
    "User":    User{},
    "Post":    Post{},
    "Comment": Comment{},
    // Add all your models here
}
```

### 5. Initialize and generate migrations

```bash
go run cmd/migration/main.go init
go run cmd/migration/main.go register [path/to/models]
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
    "github.com/beesaferoot/gorm-schema/migration"
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

## Limitations

- **Index changes (add/drop/modify) are only guaranteed for new tables.**
  - If you add, remove, or modify indexes on existing tables, these changes may not be automatically generated in migration files. You must add such index changes manually to your migrations.
- **Foreign key diffs are currently ignored.**
  - Changes to foreign key constraints (add/drop/modify) are not detected or generated in migrations.
- **Schema comparison is model-driven.**
  - Only columns present in your Go models are considered for schema diffs. Any manual changes to the database schema that are not reflected in your models will not be detected.

If you require support for these features, please open an issue or contribute to the project.

## License

MIT License - see [LICENSE](LICENSE) file for details.
