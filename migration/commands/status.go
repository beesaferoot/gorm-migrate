package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/beesaferoot/gorm-schema/migration"
)

func StatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			fmt.Printf("%-16s  %-30s  %-8s\n", "Version", "Name", "Status")
			for _, migration := range migrations {
				status := "Pending"
				if appliedMap[migration.Version] {
					status = "Applied"
				}
				fmt.Printf("%-16s  %-30s  %-8s\n", migration.Version, migration.Name, status)
			}

			return nil
		},
	}

	cmd.Flags().Bool("debug", false, "Enable debug output")

	return cmd
}
