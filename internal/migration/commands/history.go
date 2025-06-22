package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/beesaferoot/gorm-schema/internal/migration"
)

func HistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "Show migration history",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := getDB()
			if err != nil {
				return err
			}

			var records []migration.MigrationRecord
			if err := db.Order("applied_at DESC").Find(&records).Error; err != nil {
				return fmt.Errorf("failed to get migration history: %v", err)
			}

			if len(records) == 0 {
				fmt.Println("No migrations have been applied yet.")
				return nil
			}

			fmt.Printf("%-16s  %-30s  %-24s\n", "Version", "Name", "Applied At")
			for _, record := range records {
				fmt.Printf("%-16s  %-30s  %-24s\n", record.Version, record.Name, record.AppliedAt.Format(time.RFC3339))
			}

			return nil
		},
	}
}
