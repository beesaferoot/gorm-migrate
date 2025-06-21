package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"gorm-schema/internal/migration"
)

func UpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			debug, _ := cmd.Flags().GetBool("debug")

			db, err := getDB()
			if err != nil {
				return err
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

			var records []migration.MigrationRecord
			if err := db.Find(&records).Error; err != nil {
				return fmt.Errorf("failed to get applied migrations: %v", err)
			}

			appliedMap := make(map[string]bool)
			for _, record := range records {
				appliedMap[record.Version] = true
			}

			var pending []*migration.Migration
			for _, migration := range migrations {
				if !appliedMap[migration.Version] {
					pending = append(pending, migration)
				}
			}

			if len(pending) == 0 {
				fmt.Println("No pending migrations.")
				return nil
			}

			if dryRun {
				fmt.Println("Pending migrations:")
				for _, migration := range pending {
					fmt.Printf("- %s (%s)\n", migration.Name, migration.Version)
				}
				return nil
			}

			for _, mr := range pending {
				fmt.Printf("Applying migration: %s (%s)\n", mr.Name, mr.Version)

				tx := db.Begin()
				if tx.Error != nil {
					return fmt.Errorf("failed to start transaction: %v", tx.Error)
				}

				if err := mr.Up(tx); err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to apply migration %s: %v", mr.Name, err)
				}

				record := migration.MigrationRecord{
					Version:   mr.Version,
					Name:      mr.Name,
					AppliedAt: time.Now(),
				}
				if err := tx.Create(&record).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to record migration %s: %v", mr.Name, err)
				}

				if err := tx.Commit().Error; err != nil {
					return fmt.Errorf("failed to commit transaction: %v", err)
				}

				fmt.Printf("Successfully applied migration: %s\n", mr.Name)
			}

			return nil
		},
	}

	cmd.Flags().Bool("dry-run", false, "Show pending migrations without executing them")
	cmd.Flags().Bool("debug", false, "Enable debug output")

	return cmd
}
