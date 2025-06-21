# GORM Schema Migration Tool

Automatically generate database migrations from your GORM models by comparing them with the current database schema.

## Features

- 🔄 **Auto-generate migrations** from GORM model changes
- 🎯 **Smart comparison** - only generates migrations for actual changes
- 📊 **Case-insensitive** table and column name handling
- 🚀 **Zero false positives** with advanced type normalization

## Quick Start

### 1. Install

```bash
go install github.com/yourusername/gorm-schema/cmd/gorm-schema@latest
```

### 2. Set up environment

```bash
export DATABASE_URL="postgresql://user:pass@localhost:5432/your_db"
export GORM_MODELS_PATH="./models"
```

### 3. Generate model registry

```bash
gorm-schema generate-registry
```

### 4. Initialize and generate migrations

```bash
gorm-schema init
gorm-schema generate init_db
gorm-schema up
```

## Usage

```bash
# Generate model registry from your models
gorm-schema generate-registry

# Initialize migration system
gorm-schema init

# Generate migration from model changes
gorm-schema generate <name>

# Apply migrations
gorm-schema up

# Rollback last migration
gorm-schema down

# Check status
gorm-schema status
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

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    name VARCHAR(255),
    email VARCHAR(255)
);

CREATE UNIQUE INDEX idx_users_email ON users(email);
```

## Environment Variables

| Variable           | Description                   | Required                     |
| ------------------ | ----------------------------- | ---------------------------- |
| `DATABASE_URL`     | PostgreSQL connection string  | Yes                          |
| `GORM_MODELS_PATH` | Path to your models directory | Yes                          |
| `MIGRATIONS_PATH`  | Path for migration files      | No (default: `./migrations`) |

## License

MIT License - see [LICENSE](LICENSE) file for details.
