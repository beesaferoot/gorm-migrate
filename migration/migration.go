package migration

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

type Migration struct {
	Version   string
	Name      string
	CreatedAt time.Time
	Up        func(*gorm.DB) error
	Down      func(*gorm.DB) error
}

type MigrationRecord struct {
	Version   string    `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	AppliedAt time.Time `gorm:"not null"`
}

var (
	globalMigrations = make([]*Migration, 0)
	registryMutex    sync.RWMutex
)

func RegisterMigration(migration *Migration) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	globalMigrations = append(globalMigrations, migration)
}

func GetRegisteredMigrations() []*Migration {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	migrations := make([]*Migration, len(globalMigrations))
	copy(migrations, globalMigrations)
	return migrations
}

type Migrator struct {
	db         *gorm.DB
	migrations []*Migration
}

func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: GetRegisteredMigrations(),
	}
}

func (m *Migrator) Register(migration *Migration) {
	m.migrations = append(m.migrations, migration)
}

func (m *Migrator) ensureVersionTable() error {
	return m.db.AutoMigrate(&MigrationRecord{})
}

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

func (m *Migrator) Down() error {
	var lastRecord MigrationRecord
	if err := m.db.Order("applied_at DESC").First(&lastRecord).Error; err != nil {
		return err
	}

	var targetMigration *Migration
	for _, migration := range m.migrations {
		if migration.Version == lastRecord.Version {
			targetMigration = migration
			break
		}
	}

	if targetMigration == nil {
		return nil
	}

	if err := targetMigration.Down(m.db); err != nil {
		return err
	}

	return m.db.Delete(&lastRecord).Error
}

func ResetMigrations() {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	globalMigrations = make([]*Migration, 0)
}

// ModelRegistry - users must implement this
type ModelRegistry interface {
	GetModels() map[string]interface{}
}

// Global registry - users set this in their main.go
var GlobalModelRegistry ModelRegistry

// Validate that registry is provided
func ValidateRegistry() error {
	if GlobalModelRegistry == nil {
		return fmt.Errorf("no model registry provided. Please implement migration.ModelRegistry and set it in your main.go")
	}
	return nil
}
