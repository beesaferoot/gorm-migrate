package diff

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
	"sync"
)

var (
	relationshipRegistry = make(map[string]map[string]string) // model -> field -> foreignKey
	relationshipMutex    sync.RWMutex
)

// LoadModelsFromPlugin loads models from a Go plugin
func LoadModelsFromPlugin(pluginPath string) error {
	// Open the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %v", err)
	}

	// Look for the init symbol
	initSym, err := p.Lookup("init")
	if err != nil {
		return fmt.Errorf("plugin does not have an init function: %v", err)
	}

	// Call the init function
	initFunc, ok := initSym.(func())
	if !ok {
		return fmt.Errorf("init symbol is not a function")
	}
	initFunc()

	return nil
}

// RegisterRelationship registers a relationship between models
func RegisterRelationship(modelName, fieldName, foreignKey string) {
	relationshipMutex.Lock()
	defer relationshipMutex.Unlock()
	if _, exists := relationshipRegistry[modelName]; !exists {
		relationshipRegistry[modelName] = make(map[string]string)
	}
	relationshipRegistry[modelName][fieldName] = foreignKey
}

// GetModelRelationships returns all relationships for a model
func GetModelRelationships(modelName string) map[string]string {
	relationshipMutex.RLock()
	defer relationshipMutex.RUnlock()
	return relationshipRegistry[modelName]
}

// LoadModelStructs loads models from a plugin or directory
func LoadModelStructs(modelsPath string) ([]reflect.Type, error) {
	// Check if the path is a plugin file
	if filepath.Ext(modelsPath) == ".so" || filepath.Ext(modelsPath) == ".dylib" || filepath.Ext(modelsPath) == ".dll" {
		if err := LoadModelsFromPlugin(modelsPath); err != nil {
			return nil, fmt.Errorf("failed to load models from plugin: %v", err)
		}
	} else {
		// Try loading from directory (legacy support)
		if err := loadModelsFromDir(modelsPath); err != nil {
			return nil, fmt.Errorf("failed to load models from directory: %v", err)
		}
	}

	// Convert registered models to reflect.Type slice
	models := GetAllModels()
	modelTypes := make([]reflect.Type, 0, len(models))
	for _, model := range models {
		if t := reflect.TypeOf(model); t != nil {
			modelTypes = append(modelTypes, t)
		}
	}

	return modelTypes, nil
}

// loadModelsFromDir loads models from a specific directory using the real model types
func loadModelsFromDir(dir string) error {
	// Import the models package to get access to the registry
	// This is a simplified approach - in practice, we'd need to dynamically import
	// For now, let's use the existing registry approach

	// Walk the directory and parse Go files to register relationships
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		// Inspect AST for struct types to register relationships
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
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

				// Check if it's a GORM model by looking for gorm.Model or gorm tags
				isGormModel := false
				for _, field := range structType.Fields.List {
					if field.Tag != nil && field.Tag.Value != "" &&
						containsGormTag(field.Tag.Value) {
						isGormModel = true
						break
					}
					// Check for embedded gorm.Model
					if len(field.Names) == 0 {
						if sel, ok := field.Type.(*ast.SelectorExpr); ok {
							if x, ok := sel.X.(*ast.Ident); ok && x.Name == "gorm" && sel.Sel.Name == "Model" {
								isGormModel = true
								break
							}
						}
					}
				}

				if isGormModel {
					// Process relationships
					for _, field := range structType.Fields.List {
						if field.Tag != nil && field.Tag.Value != "" {
							tag := field.Tag.Value
							if strings.Contains(tag, "foreignKey:") {
								// Extract the foreign key field name
								fkField := extractForeignKeyField(tag)
								if fkField != "" {
									// Register the relationship
									RegisterRelationship(typeSpec.Name.Name, field.Names[0].Name, fkField)
								}
							}
						}
					}
				}
			}
		}
		return nil
	})

	return err
}

// containsGormTag checks if a struct tag contains `gorm:`
func containsGormTag(tag string) bool {
	return len(tag) > 7 && tag[1:6] == "gorm:"
}

// extractForeignKeyField extracts the foreign key field name from a GORM tag
func extractForeignKeyField(tag string) string {
	if !strings.Contains(tag, "foreignKey:") {
		return ""
	}
	parts := strings.Split(tag, "foreignKey:")
	if len(parts) < 2 {
		return ""
	}
	fkPart := parts[1]
	if idx := strings.Index(fkPart, "`"); idx != -1 {
		fkPart = fkPart[:idx]
	}
	return strings.TrimSpace(fkPart)
}
