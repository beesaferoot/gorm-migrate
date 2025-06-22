package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func GenerateRegistryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-registry",
		Short: "Generate model registry from GORM models",
		RunE: func(cmd *cobra.Command, args []string) error {
			modelsPath, _ := cmd.Flags().GetString("models-path")
			outputPath, _ := cmd.Flags().GetString("output")

			if modelsPath != "" {
				if err := os.Setenv("GORM_MODELS_PATH", modelsPath); err != nil {
					return fmt.Errorf("failed to set GORM_MODELS_PATH: %v", err)
				}
			}
			if outputPath != "" {
				if err := os.Setenv("GORM_MODELS_REGISTRY_FILE", outputPath); err != nil {
					return fmt.Errorf("failed to set GORM_MODELS_REGISTRY_FILE: %v", err)
				}
			}

			fmt.Println("Generating model registry...")

			execCmd := exec.Command("go", "run", "tools/gen_models_registry.go")
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Dir = "."

			if err := execCmd.Run(); err != nil {
				return fmt.Errorf("failed to generate model registry: %v", err)
			}

			fmt.Println("Model registry generated successfully!")
			return nil
		},
	}

	cmd.Flags().String("models-path", "", "Path to models directory (defaults to GORM_MODELS_PATH env var)")
	cmd.Flags().String("output", "", "Output file path (defaults to GORM_MODELS_REGISTRY_FILE env var)")

	return cmd
}
