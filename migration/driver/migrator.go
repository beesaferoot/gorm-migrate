package driver

import (
	"github.com/beesaferoot/gorm-schema/migration"
	"time"

	"gorm.io/gorm"
)

// Migrator handles the execution of migrations
type Migrator struct {
	db         *gorm.DB
	migrations []*migration.Migration
}

// NewMigrator creates a new Migrator instance
func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: make([]*migration.Migration, 0),
	}
}

// Register adds a migration to the migrator
func (m *Migrator) Register(migration *migration.Migration) {
	m.migrations = append(m.migrations, migration)
}

// ensureVersionTable creates the version tracking table if it doesn't exist
func (m *Migrator) ensureVersionTable() error {
	return m.db.AutoMigrate(&migration.MigrationRecord{})
}

// GetAppliedVersions returns a map of applied migration versions
func (m *Migrator) GetAppliedVersions() (map[string]bool, error) {
	if err := m.ensureVersionTable(); err != nil {
		return nil, err
	}

	var records []migration.MigrationRecord
	if err := m.db.Find(&records).Error; err != nil {
		return nil, err
	}

	versions := make(map[string]bool)
	for _, record := range records {
		versions[record.Version] = true
	}
	return versions, nil
}

// Up applies all pending migrations
func (m *Migrator) Up() error {
	applied, err := m.GetAppliedVersions()
	if err != nil {
		return err
	}

	for _, mr := range m.migrations {
		if !applied[mr.Version] {
			if err := mr.Up(m.db); err != nil {
				return err
			}

			record := migration.MigrationRecord{
				Version:   mr.Version,
				Name:      mr.Name,
				AppliedAt: time.Now(),
			}

			if err := m.db.Create(&record).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// Down rolls back the last applied migration
func (m *Migrator) Down() error {
	var lastRecord migration.MigrationRecord
	if err := m.db.Order("applied_at DESC").First(&lastRecord).Error; err != nil {
		return err
	}

	// Find the corresponding migration
	var targetMigration *migration.Migration
	for _, mr := range m.migrations {
		if mr.Version == lastRecord.Version {
			targetMigration = mr
			break
		}
	}

	if targetMigration == nil {
		return nil // Migration not found, might have been deleted
	}

	// Execute the down migration
	if err := targetMigration.Down(m.db); err != nil {
		return err
	}

	// Remove the migration record
	return m.db.Delete(&lastRecord).Error
}
