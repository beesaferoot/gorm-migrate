package file

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/beesaferoot/gorm-schema/migration"

	"gorm.io/gorm"
)

// LoadMigrations loads all registered migrations by importing migration files
func (l *MigrationLoader) LoadMigrations() ([]*migration.Migration, error) {
	// TEST HOOK: If TEST_MIGRATION_REGISTRY_ONLY is set, just return the global registry
	if os.Getenv("TEST_MIGRATION_REGISTRY_ONLY") == "1" {
		migrations := migration.GetRegisteredMigrations()
		sort.Slice(migrations, func(i, j int) bool {
			return migrations[i].Version < migrations[j].Version
		})
		return migrations, nil
	}

	// Check if migrations directory exists
	if _, err := os.Stat(l.directory); os.IsNotExist(err) {
		// No migrations directory, return empty list
		return []*migration.Migration{}, nil
	}

	// Check if there are any migration files
	files, err := os.ReadDir(l.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter for .go files
	var goFiles []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			goFiles = append(goFiles, file)
		}
	}

	// Only import migration files if there are any
	if len(goFiles) > 0 {
		if err := l.importMigrationFiles(); err != nil {
			return nil, fmt.Errorf("failed to import migration files: %w", err)
		}
	}

	// Get all registered migrations
	migrations := migration.GetRegisteredMigrations()

	// Sort by version (ascending)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// importMigrationFiles imports all Go files in the migrations directory
func (l *MigrationLoader) importMigrationFiles() error {
	// Read all .go files in the migrations directory
	files, err := os.ReadDir(l.directory)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Parse each migration file to extract migration information
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			filePath := filepath.Join(l.directory, file.Name())
			if err := l.parseMigrationFile(filePath); err != nil {
				return fmt.Errorf("failed to parse migration file %s: %w", file.Name(), err)
			}
		}
	}

	return nil
}

// parseMigrationFile parses a single migration file to extract migration information
func (l *MigrationLoader) parseMigrationFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Extract version and name from filename
	fileName := filepath.Base(filePath)
	parts := strings.Split(strings.TrimSuffix(fileName, ".go"), "_")
	if len(parts) < 2 {
		return fmt.Errorf("invalid migration filename format: %s", fileName)
	}

	version := parts[0]
	name := strings.Join(parts[1:], "_")

	// Create migration object
	migrationObj := &migration.Migration{
		Version:   version,
		Name:      name,
		CreatedAt: time.Now(), // We don't have the exact creation time from the file
		Up: func(db *gorm.DB) error {
			return l.executeMigrationSQL(db, string(content), "Up")
		},
		Down: func(db *gorm.DB) error {
			return l.executeMigrationSQL(db, string(content), "Down")
		},
	}

	// Register the migration
	migration.RegisterMigration(migrationObj)

	return nil
}

// executeMigrationSQL executes SQL statements from migration file content
func (l *MigrationLoader) executeMigrationSQL(db *gorm.DB, content, function string) error {
	// Parse the content to extract SQL statements from the specified function
	statements, err := l.extractSQLFromFunction(content, function)
	if err != nil {
		return fmt.Errorf("failed to extract SQL from %s function: %w", function, err)
	}

	// Execute each SQL statement
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("failed to execute SQL: %w", err)
		}
	}

	return nil
}

// extractSQLFromFunction extracts SQL statements from a specific function in the migration file
func (l *MigrationLoader) extractSQLFromFunction(content, function string) ([]string, error) {
	var statements []string

	lines := strings.Split(content, "\n")
	inFunction := false

	if l.debug {
		fmt.Printf("[DEBUG] Looking for %s function\n", function)
	}

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering the target function (anonymous function pattern)
		if strings.Contains(line, fmt.Sprintf("%s: func(db *gorm.DB)", function)) {
			inFunction = true
			if l.debug {
				fmt.Printf("[DEBUG] Entering %s function at line %d\n", function, i+1)
			}
			continue
		}

		// Check if we're exiting the function
		if inFunction && (strings.Contains(line, "return nil") || strings.Contains(line, "},")) {
			if l.debug {
				fmt.Printf("[DEBUG] Exiting %s function at line %d\n", function, i+1)
			}
			break
		}

		// Look for db.Exec calls
		if inFunction && strings.Contains(line, "db.Exec") {
			if l.debug {
				fmt.Printf("[DEBUG] Found db.Exec at line %d: %s\n", i+1, trimmedLine)
			}

			// Extract SQL from this line and continue to next lines if needed
			sql, _, err := l.extractMultiLineSQL(lines, i)
			if err != nil {
				return nil, fmt.Errorf("failed to extract SQL at line %d: %w", i+1, err)
			}

			if sql != "" {
				if l.debug {
					fmt.Printf("[DEBUG] Extracted SQL: %s\n", sql)
				}
				statements = append(statements, sql)
			}
		}
	}

	if l.debug {
		fmt.Printf("[DEBUG] Found %d SQL statements in %s function\n", len(statements), function)
	}
	return statements, nil
}

// extractMultiLineSQL extracts SQL from a db.Exec call that may span multiple lines
func (l *MigrationLoader) extractMultiLineSQL(lines []string, startLine int) (string, bool, error) {
	var sql strings.Builder
	backtickCount := 0
	inSQL := false

	for i := startLine; i < len(lines); i++ {
		line := lines[i]

		for j, char := range line {
			if char == '`' {
				backtickCount++

				if backtickCount == 1 {
					// First backtick - start collecting SQL
					inSQL = true
					continue
				} else if backtickCount == 2 {
					// Second backtick - end of SQL
					inSQL = false
					if l.debug {
						fmt.Printf("[DEBUG] SQL extraction complete at line %d, char %d\n", i+1, j+1)
					}
					return sql.String(), true, nil
				}
			}

			if inSQL {
				sql.WriteRune(char)
			}
		}

		// If we're in SQL and the line doesn't end with a backtick, add a newline
		if inSQL && backtickCount == 1 {
			sql.WriteString("\n")
		}
	}

	// If we get here, we didn't find a closing backtick
	if inSQL {
		return "", false, fmt.Errorf("unclosed backtick in SQL at line %d", startLine+1)
	}

	return "", false, nil
}
