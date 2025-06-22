package schema

import (
	"reflect"

	GORMSchema "gorm.io/gorm/schema"
)

// Column represents a gorm field
type Column struct {
	Field *GORMSchema.Field
}

func (c *Column) Type() string {
	return string(c.Field.DataType)
}

func (c *Column) ColumnName() string {
	return c.Field.DBName
}

func (c *Column) ColumnBindNames() []string {
	return c.Field.BindNames
}

func (c *Column) ColumnTag() reflect.StructTag {
	return c.Field.Tag
}
