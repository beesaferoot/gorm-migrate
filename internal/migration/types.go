package migration

import (
	"sync"
	"time"

	"gorm.io/gorm"
)

// Migration represents a single database migration
type Migration struct {
	Version   string    // Unique version identifier (e.g., timestamp)
	Name      string    // Human-readable name of the migration
	CreatedAt time.Time // When the migration was created
	Up        func(*gorm.DB) error
	Down      func(*gorm.DB) error
}

// MigrationRecord represents a record of an applied migration
type MigrationRecord struct {
	Version   string    `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	AppliedAt time.Time `gorm:"not null"`
}

// Global migration registry
var (
	globalMigrations = make([]*Migration, 0)
	registryMutex    sync.RWMutex
)

// RegisterMigration registers a migration globally
func RegisterMigration(migration *Migration) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	globalMigrations = append(globalMigrations, migration)
}

// GetRegisteredMigrations returns all registered migrations
func GetRegisteredMigrations() []*Migration {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	migrations := make([]*Migration, len(globalMigrations))
	copy(migrations, globalMigrations)
	return migrations
}

// Migrator handles the execution of migrations
type Migrator struct {
	db         *gorm.DB
	migrations []*Migration
}

// NewMigrator creates a new Migrator instance
func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: GetRegisteredMigrations(),
	}
}

// Register adds a migration to the migrator
func (m *Migrator) Register(migration *Migration) {
	m.migrations = append(m.migrations, migration)
}

// ensureVersionTable creates the version tracking table if it doesn't exist
func (m *Migrator) ensureVersionTable() error {
	return m.db.AutoMigrate(&MigrationRecord{})
}

// GetAppliedVersions returns a map of applied migration versions
func (m *Migrator) GetAppliedVersions() (map[string]bool, error) {
	if err := m.ensureVersionTable(); err != nil {
		return nil, err
	}

	var records []MigrationRecord
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

	for _, migration := range m.migrations {
		if !applied[migration.Version] {
			if err := migration.Up(m.db); err != nil {
				return err
			}

			record := MigrationRecord{
				Version:   migration.Version,
				Name:      migration.Name,
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
	var lastRecord MigrationRecord
	if err := m.db.Order("applied_at DESC").First(&lastRecord).Error; err != nil {
		return err
	}

	// Find the corresponding migration
	var targetMigration *Migration
	for _, migration := range m.migrations {
		if migration.Version == lastRecord.Version {
			targetMigration = migration
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

// ResetMigrations clears the global migration registry (for testing)
func ResetMigrations() {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	globalMigrations = make([]*Migration, 0)
}
