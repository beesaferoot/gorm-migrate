package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func ValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate all migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			loader, err := getMigrationLoader()
			if err != nil {
				return fmt.Errorf("failed to create migration loader: %v", err)
			}

			_, err = loader.LoadMigrations()
			if err != nil {
				return fmt.Errorf("validation failed: %v", err)
			}

			fmt.Println("All migrations are valid")
			return nil
		},
	}
}
