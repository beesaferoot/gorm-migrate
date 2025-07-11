package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/beesaferoot/gorm-schema/migration/file"
)

func getDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL not set in environment or .env file")
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func validateMigrationsPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid migrations path: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}

	if !strings.HasPrefix(absPath, wd) {
		return "", fmt.Errorf("migrations path must be within working directory")
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("migrations path is not writable: %v", err)
	}

	return absPath, nil
}

func getMigrationsDir() string {
	dir := os.Getenv("MIGRATIONS_PATH")
	if dir == "" {
		dir = "migrations"
	}

	cleanDir, err := validateMigrationsPath(dir)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		fmt.Println("Falling back to default 'migrations' directory")
		cleanDir, _ = validateMigrationsPath("migrations")
	}

	return cleanDir
}

func getMigrationLoader() (*file.MigrationLoader, error) {
	template := &file.MigrationTemplate{
		Version: "20060102150405",
		Name:    "%s",
	}
	return file.NewMigrationLoader(getMigrationsDir(), template), nil
}

func validateModelPath(path string) (string, error) {
	if path == "" {
		path = "models"
	}

	cleanpath := filepath.Clean(path)

	absPath, err := filepath.Abs(cleanpath)
	if err != nil {
		return "", fmt.Errorf("invalid model path: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	if !strings.HasPrefix(absPath, wd) {
		return "", fmt.Errorf("model path must be within working directory")
	}

	return absPath, nil
}

func createModelRegisterFile(dirPath string) (string, error) {
	filePath := filepath.Join(dirPath, "models_registry.go")

	packageName := filepath.Base(dirPath)
	allModels, err := getModels(dirPath)

	if err != nil {
		return "", err
	}

	content := fmt.Sprintf(`package %s

var ModelTypeRegistry = map[string]interface{}{
	%s
}	`, packageName, allModels)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to create model registry file: %w", err)
	}

	return filePath, nil
}

func getModels(dirPath string) (string, error) {
	var allModels []string

	files, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") || file.Name() == "models_registry.go" {
			continue
		}
		filePath := filepath.Join(dirPath, file.Name())
		modelNames, err := modelPerser(filePath)

		if err != nil {
			fmt.Printf("Warning: could not parse models from %s: %v\n", file.Name(), err)
			continue
		}
		allModels = append(allModels, modelNames...)
	}

	var contentBuilder strings.Builder
	for _, name := range allModels {
		contentBuilder.WriteString(fmt.Sprintf("\t\"%s\": %s{},\n", name, name))
	}
	return contentBuilder.String(), nil
}

func modelPerser(file string) ([]string, error) {
	var modelNames []string

	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)

		if !ok || genDecl.Tok != token.TYPE {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)

			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					if selfExpr, ok := field.Type.(*ast.SelectorExpr); ok {
						if indent, ok := selfExpr.X.(*ast.Ident); ok && indent.Name == "gorm" && selfExpr.Sel.Name == "Model" {
							modelNames = append(modelNames, typeSpec.Name.Name)
							break
						}
					}
				}
			}
		}
		return true
	})
	return modelNames, nil
}
