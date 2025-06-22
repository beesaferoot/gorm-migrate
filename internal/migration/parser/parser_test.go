package parser

import (
	"reflect"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gorm-schema/internal/migration"
)

// TestModel is a simple test model
type TestModel struct {
	ID   int    `gorm:"primaryKey"`
	Name string `gorm:"not null"`
}

// MockModelRegistry implements ModelRegistry for testing
type MockModelRegistry struct{}

func (r *MockModelRegistry) GetModelTypes() map[string]reflect.Type {
	return map[string]reflect.Type{
		"TestModel": reflect.TypeOf(TestModel{}),
	}
}

func TestNewModelParser(t *testing.T) {
	// Set up test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Test with no registry set
	migration.GlobalModelRegistry = nil
	_, err = NewModelParser(db)
	if err == nil {
		t.Error("Expected error when no registry is set")
	}

	// Test with registry set
	migration.GlobalModelRegistry = &MockModelRegistry{}
	parser, err := NewModelParser(db)
	if err != nil {
		t.Errorf("Expected no error when registry is set, got: %v", err)
	}

	if parser == nil {
		t.Error("Expected parser to be created")
		return
	}

	if len(parser.modelTypes) != 1 {
		t.Errorf("Expected 1 model type, got %d", len(parser.modelTypes))
	}

	if _, exists := parser.modelTypes["TestModel"]; !exists {
		t.Error("Expected TestModel to be in parser")
	}
}
