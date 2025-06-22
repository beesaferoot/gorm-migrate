package schema

import (
	"sync"

	GORMSchema "gorm.io/gorm/schema"
)

// Table represents a gorm model
type Table struct {
	*GORMSchema.Schema
	Columns []*Column
}

func (t *Table) TableName() string {
	return t.Table
}

func (t *Table) TableColumns() []*Column {
	return t.Columns
}

func CreateTableFromModel(model interface{}) (*Table, error) {
	modelSchema, err := GORMSchema.Parse(model, &sync.Map{}, GORMSchema.NamingStrategy{})
	if err != nil {
		return nil, err
	}

	columns := make([]*Column, 0)

	for _, field := range modelSchema.Fields {
		column := &Column{
			Field: field,
		}
		columns = append(columns, column)
	}

	return &Table{Schema: modelSchema, Columns: columns}, nil
}
