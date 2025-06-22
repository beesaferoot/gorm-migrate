package diff

import (
	"sync"
)

// ModelRegistry stores all registered model types
type ModelRegistry struct {
	models map[string]interface{}
	mu     sync.RWMutex
}

var (
	registry = &ModelRegistry{
		models: make(map[string]interface{}),
	}
)

// RegisterModel registers a model type with the registry
func RegisterModel(name string, model interface{}) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.models[name] = model
}

// GetModel returns a model instance by name
func GetModel(name string) (interface{}, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	model, ok := registry.models[name]
	return model, ok
}

// GetAllModels returns all registered models
func GetAllModels() []interface{} {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	models := make([]interface{}, 0, len(registry.models))
	for _, model := range registry.models {
		models = append(models, model)
	}
	return models
}
