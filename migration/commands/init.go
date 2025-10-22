package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/beesaferoot/gorm-migrate/migration"
)

func InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize migration tracking table in the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := getDB()
			if err != nil {
				return err
			}

			migrationsDir, err := validateMigrationsPath(getMigrationsDir())
			if err != nil {
				return fmt.Errorf("failed to validate migrations directory: %v", err)
			}

			if err := os.MkdirAll(migrationsDir, 0755); err != nil {
				return fmt.Errorf("failed to create migrations directory: %v", err)
			}

			if err := db.AutoMigrate(&migration.MigrationRecord{}); err != nil {
				return fmt.Errorf("failed to create schema_migrations table: %v", err)
			}

			fmt.Printf("Migration system initialized successfully in %s\n", migrationsDir)
			return nil
		},
	}
}
