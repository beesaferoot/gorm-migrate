package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/beesaferoot/gorm-schema/internal/migration/diff"
	"github.com/beesaferoot/gorm-schema/internal/migration/generator"
	modelparser "github.com/beesaferoot/gorm-schema/internal/migration/parser"
)

func GenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate [name]",
		Short: "Generate a migration from model changes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			db, err := getDB()
			if err != nil {
				return err
			}

			parser, err := modelparser.NewModelParser(db)
			if err != nil {
				return fmt.Errorf("failed to create model parser: %v", err)
			}

			modelSchemas, err := parser.Parse()
			if err != nil {
				return fmt.Errorf("failed to parse models: %v", err)
			}

			if len(modelSchemas) == 0 {
				return fmt.Errorf("no GORM models found in registry")
			}

			comparer := diff.NewSchemaComparer(db)

			currentSchema, err := comparer.GetCurrentSchema()
			if err != nil {
				return fmt.Errorf("failed to get current schema: %v", err)
			}

			changes, err := comparer.CompareSchemas(currentSchema, modelSchemas)
			if err != nil {
				return fmt.Errorf("failed to compare schemas: %v", err)
			}

			if changes == nil {
				fmt.Println("No schema changes detected")
				return nil
			}

			if !hasChanges(changes) {
				fmt.Println("No schema changes detected")
				return nil
			}

			gen := generator.NewGenerator(getMigrationsDir())
			gen.SetSchemaDiff(changes)

			if err := gen.CreateMigration(name); err != nil {
				return fmt.Errorf("failed to generate migration: %v", err)
			}

			fmt.Printf("Generated migration: %s\n", name)
			return nil
		},
	}
}

func hasChanges(changes *diff.SchemaDiff) bool {
	if len(changes.TablesToCreate) > 0 || len(changes.TablesToDrop) > 0 || len(changes.TablesToRename) > 0 {
		return true
	}

	for _, tableDiff := range changes.TablesToModify {
		if !tableDiff.IsEmpty() {
			return true
		}
	}

	return false
}
