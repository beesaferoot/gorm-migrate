package file

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// MigrationFile represents a single migration file
type MigrationFile struct {
	Path      string    // Full path to the migration file
	Version   string    // Migration version (e.g., timestamp)
	Name      string    // Migration name
	CreatedAt time.Time // When the file was created
	Up        func(*gorm.DB) error
	Down      func(*gorm.DB) error
}

// MigrationTemplate defines the format for migration files
type MigrationTemplate struct {
	Version string // Time format for version numbers
	Name    string // Format string for migration names
}

// MigrationRecord represents a record of an applied migration
type MigrationRecord struct {
	ID        uint      `gorm:"primarykey"`
	Version   string    `gorm:"uniqueIndex"`
	Name      string    `gorm:"index"`
	AppliedAt time.Time `gorm:"index"`
}

// MigrationLoader handles loading and managing migration files
type MigrationLoader struct {
	directory string
	template  *MigrationTemplate
	debug     bool
}

// NewMigrationLoader creates a new migration loader
func NewMigrationLoader(directory string, template *MigrationTemplate) *MigrationLoader {
	if template == nil {
		template = &MigrationTemplate{
			Version: "20060102150405",
			Name:    "%s",
		}
	}
	return &MigrationLoader{
		directory: directory,
		template:  template,
		debug:     false, // Debug disabled by default
	}
}

// SetDebug enables or disables debug output
func (l *MigrationLoader) SetDebug(debug bool) {
	l.debug = debug
}

// FormatName formats a migration name according to the template
func (t *MigrationTemplate) FormatName(name string) string {
	if t.Name == "" {
		return name
	}
	return fmt.Sprintf(t.Name, name)
}
