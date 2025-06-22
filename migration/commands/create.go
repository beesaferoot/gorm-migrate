package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func CreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			version := time.Now().Format("20060102150405")
			filename := fmt.Sprintf("%s_%s.go", version, name)

			migrationsDir, err := validateMigrationsPath(getMigrationsDir())
			if err != nil {
				return fmt.Errorf("failed to validate migrations directory: %v", err)
			}

			if err := os.MkdirAll(migrationsDir, 0755); err != nil {
				return fmt.Errorf("failed to create migrations directory: %v", err)
			}

			content := fmt.Sprintf(`package migrations

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if err := db.Exec(` + "`" + `CREATE TABLE example (
		id SERIAL PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)` + "`" + `).Error; err != nil {
		return err
	}

	if err := db.Exec(` + "`" + `DROP TABLE example` + "`" + `).Error; err != nil {
		return err
	}

	return nil
}`)

			filePath := filepath.Join(migrationsDir, filename)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create migration file: %v", err)
			}

			fmt.Printf("Created migration: %s\n", filePath)
			return nil
		},
	}
}
