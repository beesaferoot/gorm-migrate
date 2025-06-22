package migration

import (
	"reflect"
	"testing"
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

func TestValidateRegistry(t *testing.T) {
	// Test with no registry set
	GlobalModelRegistry = nil
	err := ValidateRegistry()
	if err == nil {
		t.Error("Expected error when no registry is set")
	}

	// Test with registry set
	GlobalModelRegistry = &MockModelRegistry{}
	err = ValidateRegistry()
	if err != nil {
		t.Errorf("Expected no error when registry is set, got: %v", err)
	}
}

func TestModelRegistry(t *testing.T) {
	registry := &MockModelRegistry{}
	modelTypes := registry.GetModelTypes()

	if len(modelTypes) != 1 {
		t.Errorf("Expected 1 model type, got %d", len(modelTypes))
	}

	if _, exists := modelTypes["TestModel"]; !exists {
		t.Error("Expected TestModel to be in registry")
	}

	expectedType := reflect.TypeOf(TestModel{})
	if modelTypes["TestModel"] != expectedType {
		t.Errorf("Expected type %v, got %v", expectedType, modelTypes["TestModel"])
	}
}
