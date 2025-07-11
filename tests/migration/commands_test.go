package migration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/beesaferoot/gorm-schema/migration"
	"github.com/beesaferoot/gorm-schema/migration/commands"
)

func TestRegisterCmd(t *testing.T) {
	cmd := commands.RegisterCmd()
	assert.Equal(t, "register [path]", cmd.Use)
	assert.Equal(t, "Generates model registry file", cmd.Short)
	assert.Equal(t, `Scans the given path for Go files containing GORM models (structs embedding gorm.Model) and generates a models_registry.go file. If no path is provided, it defaults to the 'models' directory.`, cmd.Long)
}

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

	defer func() {
		if originalDBURL != "" {
			if err := os.Setenv("DATABASE_URL", originalDBURL); err != nil {
				t.Errorf("failed to restore DATABASE_URL: %v", err)
			}
		} else {
			if err := os.Unsetenv("DATABASE_URL"); err != nil {
				t.Errorf("failed to unset DATABASE_URL: %v", err)
			}
		}
	}()

	if err := os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test"); err != nil {
		t.Errorf("failed to set DATABASE_URL: %v", err)
	}

	assert.Equal(t, "postgres://test:test@localhost:5432/test", os.Getenv("DATABASE_URL"))
}
