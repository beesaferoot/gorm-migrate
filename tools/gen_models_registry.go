package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env if available
	_ = godotenv.Load()

	var modelsDir string
	if len(os.Args) >= 2 {
		modelsDir = os.Args[1]
	} else {
		modelsDir = os.Getenv("GORM_MODELS_PATH")
		if modelsDir == "" {
			fmt.Println("Usage: go run gen_models_registry.go <models_dir> OR set GORM_MODELS_PATH environment variable")
			os.Exit(1)
		}
	}
	outputFile := filepath.Join(modelsDir, "models_registry.go")

	structs := []string{}

	files, err := os.ReadDir(modelsDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".go") || file.Name() == "models_registry.go" {
			continue
		}
		path := filepath.Join(modelsDir, file.Name())
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		for _, decl := range node.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); ok {
					structs = append(structs, typeSpec.Name.Name)
				}
			}
		}
	}

	// Generate the registry file
	var b strings.Builder
	b.WriteString("package models\n\n")
	b.WriteString("import \"reflect\"\n\n")
	b.WriteString("var ModelTypeRegistry = map[string]reflect.Type{\n")
	for _, name := range structs {
		b.WriteString(fmt.Sprintf("\t\"%s\": reflect.TypeOf(%s{}),\n", name, name))
	}
	b.WriteString("}\n")

	err = os.WriteFile(outputFile, []byte(b.String()), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Generated %s with %d models.\n", outputFile, len(structs))
}
