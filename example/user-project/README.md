# Example User Project

This example demonstrates how to use the GORM Schema Migration Tool with your own models.

## Project Structure

```
user-project/
├── cmd/
│   └── migration/
│       └── main.go          # Your migration binary
├── models/
│   ├── user.go              # Your GORM models
│   └── models_registry.go   # Your model registry
└── README.md
```

## How to Use

### 1. Copy the Example

Copy this example structure to your project and modify the files as needed.

### 2. Update the Import Path

In `cmd/migration/main.go`, change the import path:

```go
import (
    "your-project/models" // ← Change this to your actual models package
)
```

### 3. Update Your Models

In `models/user.go`, define your GORM models:

```go
package models

import "gorm.io/gorm"

type User struct {
    gorm.Model
    Name  string `gorm:"not null"`
    Email string `gorm:"uniqueIndex;not null"`
    Age   int
}

type Post struct {
    gorm.Model
    Title   string `gorm:"not null"`
    Content string
    UserID  uint
    User    User `gorm:"foreignKey:UserID"`
}
```

### 4. Update Your Registry

In `models/models_registry.go`, add all your models:

```go
package models

import "reflect"

var ModelTypeRegistry = map[string]reflect.Type{
    "User": reflect.TypeOf(User{}),
    "Post": reflect.TypeOf(Post{}),
    // Add more models here
}
```

### 5. Run Migrations

```bash
# Set your database URL
export DATABASE_URL="postgresql://user:pass@localhost:5432/your_db"

# Initialize migration system
go run cmd/migration/main.go init

# Generate migration from model changes
go run cmd/migration/main.go generate init_db

# Apply migrations
go run cmd/migration/main.go up
```

## Key Points

- **No Environment Variables**: You don't need to set `GORM_MODELS_PATH`
- **Type Safety**: Full Go compilation and IDE support
- **Simple Interface**: Just implement one method: `GetModelTypes()`
- **Flexible**: Works with any project structure

## Available Commands

- `init` - Initialize migration tracking table
- `generate <name>` - Generate migration from model changes
- `up` - Apply pending migrations
- `down` - Rollback last migration
- `status` - Show migration status
- `history` - Show migration history
- `validate` - Validate all migrations
