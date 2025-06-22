package schema

import (
	"reflect"

	GORMSchema "gorm.io/gorm/schema"
)

// Column represents a gorm field
type Column struct {
	*GORMSchema.Field
}

func (c *Column) Type() string {
	return string(c.DataType)
}

func (c *Column) ColumnName() string {
	return c.DBName
}

func (c *Column) ColumnBindNames() []string {
	return c.BindNames
}

func (c *Column) ColumnTag() reflect.StructTag {
	return c.Tag
}
