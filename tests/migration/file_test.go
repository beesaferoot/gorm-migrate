package migration

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/beesaferoot/gorm-migrate/migration"
	"github.com/beesaferoot/gorm-migrate/migration/file"
)

func TestMigrationFile(t *testing.T) {
	// Set test hook to use registry only
	os.Setenv("TEST_MIGRATION_REGISTRY_ONLY", "1")
	defer func() {
		if err := os.Unsetenv("TEST_MIGRATION_REGISTRY_ONLY"); err != nil {
			t.Errorf("failed to unset TEST_MIGRATION_REGISTRY_ONLY: %v", err)
		}
	}()

	migration.ResetMigrations()
	// Create migration loader with template
	template := &file.MigrationTemplate{
		Version: "20060102150405",
		Name:    "create_%s_table",
	}
	loader := file.NewMigrationLoader("", template)

	t.Run("Register and Load Migration", func(t *testing.T) {
		version := time.Now().Format("20060102150405")
		name := "test_migration"
		upFunc := func(db *gorm.DB) error {
			return db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)").Error
		}
		downFunc := func(db *gorm.DB) error {
			return db.Exec("DROP TABLE test_table").Error
		}
		migration.RegisterMigration(&migration.Migration{
			Version:   version,
			Name:      name,
			CreatedAt: time.Now(),
			Up:        upFunc,
			Down:      downFunc,
		})

		migrations, err := loader.LoadMigrations()
		require.NoError(t, err)

		var found *migration.Migration
		for _, m := range migrations {
			if m.Name == name {
				found = m
				break
			}
		}
		require.NotNil(t, found, "test_migration not found")
		assert.Equal(t, version, found.Version)
		assert.Equal(t, name, found.Name)
		assert.NotNil(t, found.Up)
		assert.NotNil(t, found.Down)
	})

	t.Run("Validate Migration Not Registered", func(t *testing.T) {
		migration.ResetMigrations()
		loader := file.NewMigrationLoader("", template)
		migrations, err := loader.LoadMigrations()
		require.NoError(t, err)
		var found *migration.Migration
		for _, m := range migrations {
			if m.Name == "invalid_migration" {
				found = m
				break
			}
		}
		assert.Nil(t, found, "invalid migration should not be found")
	})
}
