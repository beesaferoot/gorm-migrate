package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/beesaferoot/gorm-schema/internal/migration/file"
)

func getDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL not set in environment or .env file")
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func validateMigrationsPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid migrations path: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}

	if !strings.HasPrefix(absPath, wd) {
		return "", fmt.Errorf("migrations path must be within working directory")
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("migrations path is not writable: %v", err)
	}

	return absPath, nil
}

func getMigrationsDir() string {
	dir := os.Getenv("MIGRATIONS_PATH")
	if dir == "" {
		dir = "migrations"
	}

	cleanDir, err := validateMigrationsPath(dir)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		fmt.Println("Falling back to default 'migrations' directory")
		cleanDir, _ = validateMigrationsPath("migrations")
	}

	return cleanDir
}

func getMigrationLoader() (*file.MigrationLoader, error) {
	template := &file.MigrationTemplate{
		Version: "20060102150405",
		Name:    "%s",
	}
	return file.NewMigrationLoader(getMigrationsDir(), template), nil
}
