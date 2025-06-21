package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"gorm-schema/internal/migration"
	"gorm-schema/internal/migration/diff"
	"gorm-schema/internal/migration/file"
	"gorm-schema/internal/migration/generator"
	modelparser "gorm-schema/internal/migration/parser"
)

func main() {
	// Load .env file if present
	_ = godotenv.Load()

	var rootCmd = &cobra.Command{
		Use:   "gorm-schema",
		Short: "GORM Schema & Migration Tool",
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(generateRegistryCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// getDB returns a database connection
func getDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL not set in environment or .env file")
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// Helper function to validate and clean migrations directory path
func validateMigrationsPath(path string) (string, error) {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Convert to absolute path if relative
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid migrations path: %v", err)
	}

	// Check if path is within current directory
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}

	// Ensure path is within working directory
	if !strings.HasPrefix(absPath, wd) {
		return "", fmt.Errorf("migrations path must be within working directory")
	}

	// Check if path is writable
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("migrations path is not writable: %v", err)
	}

	return absPath, nil
}

// Helper function to get migrations directory
func getMigrationsDir() string {
	dir := os.Getenv("MIGRATIONS_PATH")
	if dir == "" {
		dir = "migrations"
	}

	// Validate and clean the path
	cleanDir, err := validateMigrationsPath(dir)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		fmt.Println("Falling back to default 'migrations' directory")
		cleanDir, _ = validateMigrationsPath("migrations")
	}

	return cleanDir
}

// Helper function to get migration loader
func getMigrationLoader() (*file.MigrationLoader, error) {
	template := &file.MigrationTemplate{
		Version: "20060102150405",
		Name:    "%s",
	}
	return file.NewMigrationLoader(getMigrationsDir(), template), nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize migration tracking table in the database",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := getDB()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Get and validate migrations directory
		migrationsDir, err := validateMigrationsPath(getMigrationsDir())
		if err != nil {
			fmt.Printf("Failed to validate migrations directory: %v\n", err)
			os.Exit(1)
		}

		// Create migrations directory if it doesn't exist
		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			fmt.Printf("Failed to create migrations directory: %v\n", err)
			os.Exit(1)
		}

		// Create schema_migrations table
		if err := db.AutoMigrate(&migration.MigrationRecord{}); err != nil {
			fmt.Printf("Failed to create schema_migrations table: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Migration system initialized successfully in %s\n", migrationsDir)
	},
}

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new migration file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		version := time.Now().Format("20060102150405")
		filename := fmt.Sprintf("%s_%s.go", version, name)

		// Get and validate migrations directory
		migrationsDir, err := validateMigrationsPath(getMigrationsDir())
		if err != nil {
			fmt.Printf("Failed to validate migrations directory: %v\n", err)
			os.Exit(1)
		}

		// Create migrations directory if it doesn't exist
		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			fmt.Printf("Failed to create migrations directory: %v\n", err)
			os.Exit(1)
		}

		// Create migration file
		content := fmt.Sprintf(`package migrations

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	// Up migration
	if err := db.Exec(`+"`"+`CREATE TABLE example (
		id SERIAL PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`+"`"+`).Error; err != nil {
		return err
	}

	// down
	if err := db.Exec(`+"`"+`DROP TABLE example`+"`"+`).Error; err != nil {
		return err
	}

	return nil
}
`, name)

		filePath := filepath.Join(migrationsDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			fmt.Printf("Failed to create migration file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created migration: %s\n", filePath)
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate [name]",
	Short: "Generate a migration from model changes",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		db, err := getDB()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Get models path from environment or flag
		modelsPath := os.Getenv("GORM_MODELS_PATH")
		if modelsPath == "" {
			fmt.Println("GORM_MODELS_PATH environment variable not set")
			os.Exit(1)
		}

		// Check if directory exists
		if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
			fmt.Printf("Models directory does not exist: %s\n", modelsPath)
			os.Exit(1)
		}

		// Parse models
		parser := modelparser.NewModelParser(modelsPath, db)
		modelSchemas, err := parser.Parse()
		if err != nil {
			fmt.Printf("Failed to parse models: %v\n", err)
			os.Exit(1)
		}

		if len(modelSchemas) == 0 {
			fmt.Println("No GORM models found in registry")
			os.Exit(1)
		}

		// Create schema comparer
		comparer := diff.NewSchemaComparer(db)

		// Get current schema
		currentSchema, err := comparer.GetCurrentSchema()
		if err != nil {
			fmt.Printf("Failed to get current schema: %v\n", err)
			os.Exit(1)
		}

		// Compare schemas
		changes, err := comparer.CompareSchemas(currentSchema, modelSchemas)
		if err != nil {
			fmt.Printf("Failed to compare schemas: %v\n", err)
			os.Exit(1)
		}

		if changes == nil {
			fmt.Println("No schema changes detected")
			os.Exit(0)
		}

		// Check if there are any actual changes
		hasChanges := false

		// Check for table creation/dropping/renaming
		if len(changes.TablesToCreate) > 0 {
			hasChanges = true
		}
		if len(changes.TablesToDrop) > 0 {
			hasChanges = true
		}
		if len(changes.TablesToRename) > 0 {
			hasChanges = true
		}

		// Check for table modifications
		for _, tableDiff := range changes.TablesToModify {
			if !tableDiff.IsEmpty() {
				hasChanges = true
				break
			}
		}

		if !hasChanges {
			fmt.Println("No schema changes detected")
			os.Exit(0)
		}

		// Create migration generator
		gen := generator.NewGenerator(getMigrationsDir())

		gen.SetSchemaDiff(changes)

		// Generate migration file
		if err := gen.CreateMigration(name); err != nil {
			fmt.Printf("Failed to generate migration: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Generated migration: %s\n", name)
	},
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		debug, _ := cmd.Flags().GetBool("debug")

		db, err := getDB()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		loader, err := getMigrationLoader()
		if err != nil {
			fmt.Printf("Failed to create migration loader: %v\n", err)
			os.Exit(1)
		}

		// Enable debug mode if requested
		loader.SetDebug(debug)

		// Load all migrations
		migrations, err := loader.LoadMigrations()
		if err != nil {
			fmt.Printf("Failed to load migrations: %v\n", err)
			os.Exit(1)
		}

		// Get applied migrations
		var records []migration.MigrationRecord
		if err := db.Find(&records).Error; err != nil {
			fmt.Printf("Failed to get applied migrations: %v\n", err)
			os.Exit(1)
		}

		appliedMap := make(map[string]bool)
		for _, record := range records {
			appliedMap[record.Version] = true
		}

		// Find pending migrations
		var pending []*migration.Migration
		for _, migration := range migrations {
			if !appliedMap[migration.Version] {
				pending = append(pending, migration)
			}
		}

		if len(pending) == 0 {
			fmt.Println("No pending migrations.")
			return
		}

		if dryRun {
			fmt.Println("Pending migrations:")
			for _, migration := range pending {
				fmt.Printf("- %s (%s)\n", migration.Name, migration.Version)
			}
			return
		}

		// Apply migrations
		for _, mr := range pending {
			fmt.Printf("Applying migration: %s (%s)\n", mr.Name, mr.Version)

			// Start transaction
			tx := db.Begin()
			if tx.Error != nil {
				fmt.Printf("Failed to start transaction: %v\n", tx.Error)
				os.Exit(1)
			}

			// Execute migration
			if err := mr.Up(tx); err != nil {
				tx.Rollback()
				fmt.Printf("Failed to apply migration %s: %v\n", mr.Name, err)
				os.Exit(1)
			}

			// Record migration
			record := migration.MigrationRecord{
				Version:   mr.Version,
				Name:      mr.Name,
				AppliedAt: time.Now(),
			}
			if err := tx.Create(&record).Error; err != nil {
				tx.Rollback()
				fmt.Printf("Failed to record migration %s: %v\n", mr.Name, err)
				os.Exit(1)
			}

			// Commit transaction
			if err := tx.Commit().Error; err != nil {
				fmt.Printf("Failed to commit transaction: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Successfully applied migration: %s\n", mr.Name)
		}
	},
}

func init() {
	upCmd.Flags().Bool("dry-run", false, "Show pending migrations without executing them")
	upCmd.Flags().Bool("debug", false, "Enable debug output")
	downCmd.Flags().Bool("debug", false, "Enable debug output")
	statusCmd.Flags().Bool("debug", false, "Enable debug output")
	generateRegistryCmd.Flags().String("models-path", "", "Path to models directory (defaults to GORM_MODELS_PATH env var)")
	generateRegistryCmd.Flags().String("output", "", "Output file path (defaults to GORM_MODELS_REGISTRY_FILE env var)")
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Revert the last migration",
	Run: func(cmd *cobra.Command, args []string) {
		debug, _ := cmd.Flags().GetBool("debug")

		db, err := getDB()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Get the last applied migration
		var record migration.MigrationRecord
		if err := db.Order("applied_at DESC").First(&record).Error; err != nil {
			fmt.Println("No migrations to revert")
			os.Exit(1)
		}

		loader, err := getMigrationLoader()
		if err != nil {
			fmt.Printf("Failed to create migration loader: %v\n", err)
			os.Exit(1)
		}

		// Enable debug mode if requested
		loader.SetDebug(debug)

		// Load all migrations
		migrations, err := loader.LoadMigrations()
		if err != nil {
			fmt.Printf("Failed to load migrations: %v\n", err)
			os.Exit(1)
		}

		// Find the migration to revert
		var targetMigration *migration.Migration
		for _, m := range migrations {
			if m.Version == record.Version {
				targetMigration = m
				break
			}
		}

		if targetMigration == nil {
			fmt.Printf("Migration file for version %s not found\n", record.Version)
			os.Exit(1)
		}

		// Start transaction
		tx := db.Begin()
		if tx.Error != nil {
			fmt.Printf("Failed to start transaction: %v\n", tx.Error)
			os.Exit(1)
		}

		// Execute down migration
		if err := targetMigration.Down(tx); err != nil {
			tx.Rollback()
			fmt.Printf("Failed to revert migration %s: %v\n", targetMigration.Name, err)
			os.Exit(1)
		}

		// Remove migration record
		if err := tx.Delete(&record).Error; err != nil {
			tx.Rollback()
			fmt.Printf("Failed to remove migration record: %v\n", err)
			os.Exit(1)
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			fmt.Printf("Failed to commit transaction: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully reverted migration: %s\n", targetMigration.Name)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all migrations",
	Run: func(cmd *cobra.Command, args []string) {
		debug, _ := cmd.Flags().GetBool("debug")

		db, err := getDB()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		loader, err := getMigrationLoader()
		if err != nil {
			fmt.Printf("Failed to create migration loader: %v\n", err)
			os.Exit(1)
		}

		// Enable debug mode if requested
		loader.SetDebug(debug)

		// Load all migrations
		migrations, err := loader.LoadMigrations()
		if err != nil {
			fmt.Printf("Failed to load migrations: %v\n", err)
			os.Exit(1)
		}

		// Get applied migrations
		var records []migration.MigrationRecord
		if err := db.Find(&records).Error; err != nil {
			fmt.Printf("Failed to get applied migrations: %v\n", err)
			os.Exit(1)
		}

		appliedMap := make(map[string]bool)
		for _, record := range records {
			appliedMap[record.Version] = true
		}

		// Print status
		fmt.Printf("%-16s  %-30s  %-8s\n", "Version", "Name", "Status")
		for _, migration := range migrations {
			status := "Pending"
			if appliedMap[migration.Version] {
				status = "Applied"
			}
			fmt.Printf("%-16s  %-30s  %-8s\n", migration.Version, migration.Name, status)
		}
	},
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show migration history",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := getDB()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Get applied migrations
		var records []migration.MigrationRecord
		if err := db.Order("applied_at DESC").Find(&records).Error; err != nil {
			fmt.Printf("Failed to get migration history: %v\n", err)
			os.Exit(1)
		}

		if len(records) == 0 {
			fmt.Println("No migrations have been applied yet.")
			return
		}

		fmt.Printf("%-16s  %-30s  %-24s\n", "Version", "Name", "Applied At")
		for _, record := range records {
			fmt.Printf("%-16s  %-30s  %-24s\n", record.Version, record.Name, record.AppliedAt.Format(time.RFC3339))
		}
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate all migrations",
	Run: func(cmd *cobra.Command, args []string) {
		loader, err := getMigrationLoader()
		if err != nil {
			fmt.Printf("Failed to create migration loader: %v\n", err)
			os.Exit(1)
		}

		// Load and validate all migrations
		_, err = loader.LoadMigrations()
		if err != nil {
			fmt.Printf("Validation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("All migrations are valid")
	},
}

var generateRegistryCmd = &cobra.Command{
	Use:   "generate-registry",
	Short: "Generate model registry from GORM models",
	Run: func(cmd *cobra.Command, args []string) {
		modelsPath, _ := cmd.Flags().GetString("models-path")
		outputPath, _ := cmd.Flags().GetString("output")

		// Set environment variables for the codegen tool
		if modelsPath != "" {
			os.Setenv("GORM_MODELS_PATH", modelsPath)
		}
		if outputPath != "" {
			os.Setenv("GORM_MODELS_REGISTRY_FILE", outputPath)
		}

		// Run the codegen tool
		fmt.Println("Generating model registry...")

		// Execute the codegen tool
		cmd2 := exec.Command("go", "run", "tools/gen_models_registry.go")
		cmd2.Stdout = os.Stdout
		cmd2.Stderr = os.Stderr
		cmd2.Dir = "." // Run from current directory

		if err := cmd2.Run(); err != nil {
			fmt.Printf("Failed to generate model registry: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Model registry generated successfully!")
	},
}
