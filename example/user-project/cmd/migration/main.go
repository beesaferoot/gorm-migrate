package main

import (
	"github.com/beesaferoot/gorm-schema/example/user-project/models" // User's models package - CHANGE THIS
	"github.com/beesaferoot/gorm-schema/migration"
	"github.com/beesaferoot/gorm-schema/migration/commands"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// Simple registry implementation
type MyModelRegistry struct{}

func (r *MyModelRegistry) GetModels() map[string]interface{} {
	return models.ModelTypeRegistry // User's registry
}

func init() {
	migration.GlobalModelRegistry = &MyModelRegistry{}
}

func main() {
	_ = godotenv.Load() // optionally load environment file
	rootCmd := &cobra.Command{
		Use:   "migration",
		Short: "Database Migration Tool",
	}

	rootCmd.AddCommand(
		commands.RegisterCmd(),
		commands.InitCmd(),
		commands.CreateCmd(),
		commands.GenerateCmd(),
		commands.UpCmd(),
		commands.DownCmd(),
		commands.StatusCmd(),
		commands.HistoryCmd(),
		commands.ValidateCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
