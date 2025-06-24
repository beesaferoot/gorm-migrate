package models

// ModelTypeRegistry maps model names to their structs
var ModelTypeRegistry = map[string]interface{}{
	"User": User{},
	"Post": Post{},
}
