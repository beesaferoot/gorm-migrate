package migration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gorm-schema/internal/migration"
	"gorm-schema/internal/migration/commands"
)

func TestInitCmd(t *testing.T) {
	cmd := commands.InitCmd()
	assert.Equal(t, "init", cmd.Use)
	assert.Equal(t, "Initialize migration tracking table in the database", cmd.Short)
}

func TestCreateCmd(t *testing.T) {
	cmd := commands.CreateCmd()
	assert.Equal(t, "create [name]", cmd.Use)
	assert.Equal(t, "Create a new migration file", cmd.Short)
}

func TestGenerateCmd(t *testing.T) {
	cmd := commands.GenerateCmd()
	assert.Equal(t, "generate [name]", cmd.Use)
	assert.Equal(t, "Generate a migration from model changes", cmd.Short)
}

func TestUpCmd(t *testing.T) {
	cmd := commands.UpCmd()
	assert.Equal(t, "up", cmd.Use)
	assert.Equal(t, "Apply all pending migrations", cmd.Short)

	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("dry-run"))
	assert.NotNil(t, flags.Lookup("debug"))
}

func TestDownCmd(t *testing.T) {
	cmd := commands.DownCmd()
	assert.Equal(t, "down", cmd.Use)
	assert.Equal(t, "Revert the last migration", cmd.Short)

	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("debug"))
}

func TestStatusCmd(t *testing.T) {
	cmd := commands.StatusCmd()
	assert.Equal(t, "status", cmd.Use)
	assert.Equal(t, "Show status of all migrations", cmd.Short)

	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("debug"))
}

func TestHistoryCmd(t *testing.T) {
	cmd := commands.HistoryCmd()
	assert.Equal(t, "history", cmd.Use)
	assert.Equal(t, "Show migration history", cmd.Short)
}

func TestValidateCmd(t *testing.T) {
	cmd := commands.ValidateCmd()
	assert.Equal(t, "validate", cmd.Use)
	assert.Equal(t, "Validate all migrations", cmd.Short)
}

func TestGenerateRegistryCmd(t *testing.T) {
	cmd := commands.GenerateRegistryCmd()
	assert.Equal(t, "generate-registry", cmd.Use)
	assert.Equal(t, "Generate model registry from GORM models", cmd.Short)

	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("models-path"))
	assert.NotNil(t, flags.Lookup("output"))
}

func TestMigrationRecord(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	record := migration.MigrationRecord{
		Version:   "20231201120000",
		Name:      "test_migration",
		AppliedAt: migration.MigrationRecord{}.AppliedAt,
	}

	err = db.AutoMigrate(&migration.MigrationRecord{})
	require.NoError(t, err)

	err = db.Create(&record).Error
	require.NoError(t, err)

	var found migration.MigrationRecord
	err = db.First(&found, "version = ?", "20231201120000").Error
	require.NoError(t, err)

	assert.Equal(t, "test_migration", found.Name)
}

func TestEnvironmentVariables(t *testing.T) {
	originalDBURL := os.Getenv("DATABASE_URL")
	originalMigrationsPath := os.Getenv("MIGRATIONS_PATH")
	originalModelsPath := os.Getenv("GORM_MODELS_PATH")

	defer func() {
		if originalDBURL != "" {
			os.Setenv("DATABASE_URL", originalDBURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
		if originalMigrationsPath != "" {
			os.Setenv("MIGRATIONS_PATH", originalMigrationsPath)
		} else {
			os.Unsetenv("MIGRATIONS_PATH")
		}
		if originalModelsPath != "" {
			os.Setenv("GORM_MODELS_PATH", originalModelsPath)
		} else {
			os.Unsetenv("GORM_MODELS_PATH")
		}
	}()

	os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	os.Setenv("MIGRATIONS_PATH", "./test_migrations")
	os.Setenv("GORM_MODELS_PATH", "./test_models")

	assert.Equal(t, "postgres://test:test@localhost:5432/test", os.Getenv("DATABASE_URL"))
	assert.Equal(t, "./test_migrations", os.Getenv("MIGRATIONS_PATH"))
	assert.Equal(t, "./test_models", os.Getenv("GORM_MODELS_PATH"))
}
