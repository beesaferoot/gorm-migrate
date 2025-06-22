package models

import "reflect"

// ModelTypeRegistry maps model names to their reflect.Type
var ModelTypeRegistry = map[string]reflect.Type{
	"User": reflect.TypeOf(User{}),
	"Post": reflect.TypeOf(Post{}),
}
