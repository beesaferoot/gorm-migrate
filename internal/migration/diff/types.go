package diff

import (
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// SchemaDiff represents the differences between two database schemas
type SchemaDiff struct {
	TablesToCreate []TableDiff
	TablesToDrop   []string
	TablesToModify []TableDiff
	TablesToRename []TableRename
}

// TableDiff represents the differences in a table, using GORM types
type TableDiff struct {
	Schema            *schema.Schema
	FieldsToAdd       []*schema.Field
	FieldsToDrop      []*schema.Field
	FieldsToModify    []*schema.Field
	FieldsToRename    []ColumnRename
	IndexesToAdd      []*schema.Index
	IndexesToDrop     []*schema.Index
	IndexesToModify   []*schema.Index
	ForeignKeysToAdd  []*schema.Relationship
	ForeignKeysToDrop []*schema.Relationship
}

// IsEmpty checks if a TableDiff is empty
func (d *TableDiff) IsEmpty() bool {
	return len(d.FieldsToAdd) == 0 &&
		len(d.FieldsToModify) == 0 &&
		len(d.FieldsToDrop) == 0 &&
		len(d.IndexesToAdd) == 0 &&
		len(d.IndexesToDrop) == 0 &&
		len(d.ForeignKeysToAdd) == 0 &&
		len(d.ForeignKeysToDrop) == 0
}

// ColumnRename represents a column rename operation
type ColumnRename struct {
	OldName string
	NewName string
}

// TableRename represents a table rename operation
type TableRename struct {
	OldName string
	NewName string
}

// SchemaComparer compares database schemas
type SchemaComparer struct {
	db *gorm.DB
}

// NewSchemaComparer creates a new schema comparer
func NewSchemaComparer(db *gorm.DB) *SchemaComparer {
	return &SchemaComparer{
		db: db,
	}
}

// Compare compares the current database schema with the provided models
func (c *SchemaComparer) Compare(models ...interface{}) (*SchemaDiff, error) {
	// Get current database schema
	currentSchema, err := c.getCurrentSchema()
	if err != nil {
		return nil, err
	}

	// Get model schemas
	modelSchemas, err := c.GetModelSchemas(models...)
	if err != nil {
		return nil, err
	}

	// Compare schemas
	diff, err := c.compareSchemas(currentSchema, modelSchemas)
	if err != nil {
		return nil, err
	}

	return diff, nil
}

// Schema represents a database schema
type Schema struct {
	Tables map[string]*schema.Schema
}

// Table represents a database table
type Table struct {
	Name    string
	Schema  *schema.Schema
	Fields  map[string]*schema.Field
	Indexes []*schema.Index
}

// Column represents a database column
type Column struct {
	Name           string
	Type           string
	IsPrimaryKey   bool
	IsUnique       bool
	IsNotNull      bool
	DefaultValue   string
	Size           int
	Precision      int
	Scale          int
	Comment        string
	AutoIncrement  bool
	UniqueIndex    string
	Index          string
	ForeignKey     string
	References     string
	OnDelete       string
	OnUpdate       string
	Check          string
	Constraint     string
	Embedded       bool
	EmbeddedPrefix string
	Serializer     string
	Permission     string
	Ignore         bool
	AutoCreateTime bool
	AutoUpdateTime bool
	Tag            string
	TagSettings    map[string]string
}
