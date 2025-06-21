package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"gorm-schema/internal/migration"
)

func DownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Revert the last migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			debug, _ := cmd.Flags().GetBool("debug")

			db, err := getDB()
			if err != nil {
				return err
			}

			var record migration.MigrationRecord
			if err := db.Order("applied_at DESC").First(&record).Error; err != nil {
				return fmt.Errorf("no migrations to revert")
			}

			loader, err := getMigrationLoader()
			if err != nil {
				return fmt.Errorf("failed to create migration loader: %v", err)
			}

			loader.SetDebug(debug)

			migrations, err := loader.LoadMigrations()
			if err != nil {
				return fmt.Errorf("failed to load migrations: %v", err)
			}

			var targetMigration *migration.Migration
			for _, m := range migrations {
				if m.Version == record.Version {
					targetMigration = m
					break
				}
			}

			if targetMigration == nil {
				return fmt.Errorf("migration file for version %s not found", record.Version)
			}

			tx := db.Begin()
			if tx.Error != nil {
				return fmt.Errorf("failed to start transaction: %v", tx.Error)
			}

			if err := targetMigration.Down(tx); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to revert migration %s: %v", targetMigration.Name, err)
			}

			if err := tx.Delete(&record).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to remove migration record: %v", err)
			}

			if err := tx.Commit().Error; err != nil {
				return fmt.Errorf("failed to commit transaction: %v", err)
			}

			fmt.Printf("Successfully reverted migration: %s\n", targetMigration.Name)
			return nil
		},
	}

	cmd.Flags().Bool("debug", false, "Enable debug output")

	return cmd
} 