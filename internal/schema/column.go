package schema

import (
	GORMSchema "gorm.io/gorm/schema"
	"reflect"
)

// Column represents a gorm field
type Column struct {
	*GORMSchema.Field
}

func (c *Column) Type() string {
	return string(c.DataType)
}

func (c *Column) ColumnName() string {
	return c.Name
}

func (c *Column) ColumnBindNames() []string {
	return c.BindNames
}

func (c *Column) ColumnTag() reflect.StructTag {
	return c.Tag
}
