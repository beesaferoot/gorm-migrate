package parser

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/beesaferoot/gorm-schema/migration"
)

// TestModel is a simple test model
type TestModel struct {
	ID   int    `gorm:"primaryKey"`
	Name string `gorm:"not null"`
}

// MockModelRegistry implements ModelRegistry for testing
type MockModelRegistry struct{}

func (r *MockModelRegistry) GetModels() map[string]interface{} {
	return map[string]interface{}{
		"TestModel": TestModel{},
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

	if len(parser.models) != 1 {
		t.Errorf("Expected 1 model type, got %d", len(parser.models))
	}

	if _, exists := parser.models["TestModel"]; !exists {
		t.Error("Expected TestModel to be in parser")
	}
}
