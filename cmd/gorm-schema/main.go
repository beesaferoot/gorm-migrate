package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/beesaferoot/gorm-schema/migration/commands"
	"github.com/beesaferoot/gorm-schema/migration"
	"github.com/beesaferoot/gorm-schema/example/models"
)

type MyModelRegistry struct{}

func (r *MyModelRegistry) GetModels() map[string]interface{} {
	return models.ModelTypeRegistry
}

func init() {
	migration.GlobalModelRegistry = &MyModelRegistry{}
}

func main() {
	_ = godotenv.Load()

	rootCmd := &cobra.Command{
		Use:   "gorm-schema",
		Short: "GORM Schema & Migration Tool",
	}

	rootCmd.AddCommand(
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
