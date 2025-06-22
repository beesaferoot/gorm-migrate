package file

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/beesaferoot/gorm-schema/migration"

	"gorm.io/gorm"
)

// GenerateMigration generates a new migration file
func (l *MigrationLoader) GenerateMigration(name string, upFunc, downFunc func(*gorm.DB) error) (*MigrationFile, error) {
	// Generate version number
	version := time.Now().Format(l.template.Version)

	// Format migration name
	formattedName := l.template.FormatName(name)

	// Create filename
	filename := fmt.Sprintf("%s_%s.go", version, formattedName)
	path := filepath.Join(l.directory, filename)

	// Create migration file
	migration := &MigrationFile{
		Path:      path,
		Version:   version,
		Name:      formattedName,
		CreatedAt: time.Now(),
		Up:        upFunc,
		Down:      downFunc,
	}

	// Generate file content
	content := fmt.Sprintf(`package migrations

import (
	"gorm.io/gorm"
)

// Up applies the migration
func Up(db *gorm.DB) error {
	%s
}

// Down rolls back the migration
func Down(db *gorm.DB) error {
	%s
}
`, formatGoFunc(upFunc), formatGoFunc(downFunc))

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write migration file: %w", err)
	}

	return migration, nil
}

// formatGoFunc formats a function as a string
func formatGoFunc(fn func(*gorm.DB) error) string {
	// TODO: Implement function formatting
	// For now, return a placeholder
	return "return nil"
}

// GetPendingMigrations returns migrations that haven't been applied yet
func (l *MigrationLoader) GetPendingMigrations(db *gorm.DB) ([]*migration.Migration, error) {
	// Load all migrations
	migrations, err := l.LoadMigrations()
	if err != nil {
		return nil, err
	}

	// Get applied versions
	var appliedVersions []string
	if err := db.Table("migration_records").Pluck("version", &appliedVersions).Error; err != nil {
		return nil, fmt.Errorf("failed to get applied versions: %w", err)
	}

	// Create map of applied versions for quick lookup
	applied := make(map[string]bool)
	for _, version := range appliedVersions {
		applied[version] = true
	}

	// Filter out applied migrations
	var pending []*migration.Migration
	for _, migration := range migrations {
		if !applied[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}
