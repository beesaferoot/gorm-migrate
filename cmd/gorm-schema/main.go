package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"gorm-schema/internal/migration/commands"
)

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
		commands.GenerateRegistryCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
