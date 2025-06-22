package migration_test

import (
	"github.com/beesaferoot/gorm-schema/migration"
	"github.com/beesaferoot/gorm-schema/migration/driver"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	return db
}

func TestMigrator_Up(t *testing.T) {
	db := setupTestDB(t)
	migrator := driver.NewMigrator(db)

	// Create a test migration
	testMigration := &migration.Migration{
		Version:   "20240315000001",
		Name:      "test_migration",
		CreatedAt: time.Now(),
		Up: func(db *gorm.DB) error {
			return db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)").Error
		},
		Down: func(db *gorm.DB) error {
			return db.Exec("DROP TABLE test").Error
		},
	}

	// Register the migration
	migrator.Register(testMigration)

	// Run the migration
	err := migrator.Up()
	assert.NoError(t, err)

	// Verify the migration was applied
	var record migration.MigrationRecord
	err = db.Where("version = ?", testMigration.Version).First(&record).Error
	assert.NoError(t, err)
	assert.Equal(t, testMigration.Name, record.Name)

	// Verify the table was created
	var count int64
	err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test'").Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestMigrator_Down(t *testing.T) {
	db := setupTestDB(t)
	migrator := driver.NewMigrator(db)

	// Create a test migration
	testMigration := &migration.Migration{
		Version:   "20240315000001",
		Name:      "test_migration",
		CreatedAt: time.Now(),
		Up: func(db *gorm.DB) error {
			return db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)").Error
		},
		Down: func(db *gorm.DB) error {
			return db.Exec("DROP TABLE test").Error
		},
	}

	// Register and apply the migration
	migrator.Register(testMigration)
	err := migrator.Up()
	assert.NoError(t, err)

	// Rollback the migration
	err = migrator.Down()
	assert.NoError(t, err)

	// Verify the migration record was removed
	var record migration.MigrationRecord
	err = db.Where("version = ?", testMigration.Version).First(&record).Error
	assert.Error(t, err)

	// Verify the table was dropped
	var count int64
	err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test'").Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
