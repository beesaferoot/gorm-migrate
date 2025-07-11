package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func RegisterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "register [path]",
		Short: "Generates model registry file",
		Long:  `Scans the given path for Go files containing GORM models (structs embedding gorm.Model) and generates a models_registry.go file. If no path is provided, it defaults to the 'models' directory.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			var pathToValidate string
			if len(args) > 0 {
				pathToValidate = args[0]
			}

			validatedPath, err := validateModelPath(pathToValidate)
			if err != nil {
				return fmt.Errorf("failed to validate model path: %w", err)
			}

			//create model_register.go file:
			ModelRegistry, err := createModelRegisterFile(validatedPath)
			if err != nil {
				return fmt.Errorf("failed to create model registry file: %w", err)
			}

			fmt.Printf("Successfully generated model registry: %s\n", ModelRegistry)
			return nil

		},
	}
}
