package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"gorm-schema/internal/migration/diff"
)

// TestModels for unit testing
type TestUser struct {
	gorm.Model
	Name string
	Age  int
}

type TestUserWithNewField struct {
	gorm.Model
	Name    string
	Age     int
	Email   string // New field
	Address string // New field
}

type TestUserWithModifiedField struct {
	gorm.Model
	Name string
	Age  int64 // Changed from int to int64
}

type TestUserWithRemovedField struct {
	gorm.Model
	Name string
	// Age field removed
}

type TestUserWithRenamedField struct {
	gorm.Model
	Name    string
	UserAge int // Renamed from Age to UserAge
}

// TestSchemaComparerUnit tests the core schema comparison logic
func TestSchemaComparerUnit(t *testing.T) {
	t.Run("No Changes - Identical Schemas", func(t *testing.T) {
		// Create identical schemas
		schema1 := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", DBName: "name", DataType: "string"},
			{Name: "age", DBName: "age", DataType: "int"},
		})

		schema2 := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", DBName: "name", DataType: "string"},
			{Name: "age", DBName: "age", DataType: "int"},
		})

		comparer := diff.NewSchemaComparer(nil)
		tableDiff := comparer.CompareTable(schema1, schema2)

		assert.True(t, tableDiff.IsEmpty(), "Should detect no changes between identical schemas")
		assert.Empty(t, tableDiff.FieldsToAdd)
		assert.Empty(t, tableDiff.FieldsToModify)
		assert.Empty(t, tableDiff.FieldsToDrop)
	})

	t.Run("Add New Field", func(t *testing.T) {
		// Current schema (database)
		currentSchema := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", DBName: "name", DataType: "string"},
		})

		// Target schema (model) - has new field
		targetSchema := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", DBName: "name", DataType: "string"},
			{Name: "age", DBName: "age", DataType: "int"}, // New field
		})

		comparer := diff.NewSchemaComparer(nil)
		tableDiff := comparer.CompareTable(currentSchema, targetSchema)

		assert.False(t, tableDiff.IsEmpty(), "Should detect changes when adding new field")
		assert.Len(t, tableDiff.FieldsToAdd, 1, "Should have one field to add")
		assert.Equal(t, "age", tableDiff.FieldsToAdd[0].DBName)
		assert.Empty(t, tableDiff.FieldsToModify)
		assert.Empty(t, tableDiff.FieldsToDrop)
	})

	t.Run("Modify Field Type", func(t *testing.T) {
		// Current schema (database)
		currentSchema := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
			{Name: "age", DBName: "age", DataType: "int"},
		})

		// Target schema (model) - field type changed
		targetSchema := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
			{Name: "age", DBName: "age", DataType: "int64"}, // Changed from int to int64
		})

		comparer := diff.NewSchemaComparer(nil)
		tableDiff := comparer.CompareTable(currentSchema, targetSchema)

		// With our normalization, int and int64 should be treated as equivalent
		assert.True(t, tableDiff.IsEmpty(), "Should not detect changes for equivalent types")
		assert.Empty(t, tableDiff.FieldsToModify)
	})

	t.Run("Remove Field - Should Be Ignored", func(t *testing.T) {
		// Current schema (database) - has extra field
		currentSchema := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", DBName: "name", DataType: "string"},
			{Name: "age", DBName: "age", DataType: "int"},
			{Name: "extra", DBName: "extra", DataType: "string"}, // Extra field in DB
		})

		// Target schema (model) - doesn't have the extra field
		targetSchema := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", DBName: "name", DataType: "string"},
			{Name: "age", DBName: "age", DataType: "int"},
		})

		comparer := diff.NewSchemaComparer(nil)
		tableDiff := comparer.CompareTable(currentSchema, targetSchema)

		// Should ignore orphaned columns (fields to drop)
		assert.True(t, tableDiff.IsEmpty(), "Should ignore orphaned columns in database")
		assert.Empty(t, tableDiff.FieldsToDrop)
	})

	t.Run("Type Normalization - Equivalent Types", func(t *testing.T) {
		testCases := []struct {
			name     string
			type1    string
			type2    string
			expected bool // true if should be considered equal
		}{
			{"uint vs int8", "uint", "int8", true},
			{"int vs int4", "int", "int4", true},
			{"float64 vs float8", "float64", "float8", true},
			{"string vs varchar", "string", "varchar", true},
			{"bool vs boolean", "bool", "boolean", true},
			{"time vs timestamp", "time", "timestamp", true},
			{"json vs jsonb", "json", "jsonb", true},
			{"different types", "int", "string", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				currentSchema := createTestSchema("users", []*schema.Field{
					{Name: "id", DBName: "id", DataType: schema.DataType(tc.type1), PrimaryKey: true},
				})

				targetSchema := createTestSchema("users", []*schema.Field{
					{Name: "id", DBName: "id", DataType: schema.DataType(tc.type2), PrimaryKey: true},
				})

				comparer := diff.NewSchemaComparer(nil)
				tableDiff := comparer.CompareTable(currentSchema, targetSchema)

				if tc.expected {
					assert.True(t, tableDiff.IsEmpty(), "Equivalent types should not trigger changes")
				} else {
					assert.False(t, tableDiff.IsEmpty(), "Different types should trigger changes")
				}
			})
		}
	})

	t.Run("Case Insensitive Field Names", func(t *testing.T) {
		// Current schema (database) - lowercase field names
		currentSchema := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", DBName: "name", DataType: "string"},
		})

		// Target schema (model) - mixed case field names
		targetSchema := createTestSchema("users", []*schema.Field{
			{Name: "ID", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
			{Name: "Name", DBName: "name", DataType: "string"},
		})

		comparer := diff.NewSchemaComparer(nil)
		tableDiff := comparer.CompareTable(currentSchema, targetSchema)

		assert.True(t, tableDiff.IsEmpty(), "Should handle case-insensitive field names")
	})

	t.Run("Primary Key and AutoIncrement Handling", func(t *testing.T) {
		// Test that primary key and auto-increment fields are handled correctly
		currentSchema := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "int8", PrimaryKey: true, AutoIncrement: true, NotNull: false},
		})

		targetSchema := createTestSchema("users", []*schema.Field{
			{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true, NotNull: true},
		})

		comparer := diff.NewSchemaComparer(nil)
		tableDiff := comparer.CompareTable(currentSchema, targetSchema)

		// Should be considered equivalent due to type normalization
		assert.True(t, tableDiff.IsEmpty(), "Primary key fields should be normalized correctly")
	})
}

// TestSchemaDiffUnit tests the high-level schema diff functionality
func TestSchemaDiffUnit(t *testing.T) {
	t.Run("Empty Database vs Models", func(t *testing.T) {
		// Empty current schema
		currentSchema := make(map[string]*schema.Schema)

		// Target schema with one table
		targetSchema := map[string]*schema.Schema{
			"users": createTestSchema("users", []*schema.Field{
				{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true},
				{Name: "name", DBName: "name", DataType: "string"},
			}),
		}

		comparer := diff.NewSchemaComparer(nil)
		schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
		require.NoError(t, err)

		assert.Len(t, schemaDiff.TablesToCreate, 1, "Should detect one table to create")
		assert.Empty(t, schemaDiff.TablesToModify)
		assert.Empty(t, schemaDiff.TablesToDrop)
	})

	t.Run("Table Drop Detection", func(t *testing.T) {
		// Current schema has extra table
		currentSchema := map[string]*schema.Schema{
			"users": createTestSchema("users", []*schema.Field{
				{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true},
			}),
			"extra_table": createTestSchema("extra_table", []*schema.Field{
				{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true},
			}),
		}

		// Target schema doesn't have the extra table
		targetSchema := map[string]*schema.Schema{
			"users": createTestSchema("users", []*schema.Field{
				{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true},
			}),
		}

		comparer := diff.NewSchemaComparer(nil)
		schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
		require.NoError(t, err)

		assert.Len(t, schemaDiff.TablesToDrop, 1, "Should detect one table to drop")
		assert.Equal(t, "extra_table", schemaDiff.TablesToDrop[0])
		assert.Empty(t, schemaDiff.TablesToCreate)
		assert.Empty(t, schemaDiff.TablesToModify)
	})

	t.Run("Case Insensitive Table Names", func(t *testing.T) {
		// Current schema with lowercase table name
		currentSchema := map[string]*schema.Schema{
			"users": createTestSchema("users", []*schema.Field{
				{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true},
			}),
		}

		// Target schema with uppercase table name
		targetSchema := map[string]*schema.Schema{
			"Users": createTestSchema("Users", []*schema.Field{
				{Name: "id", DBName: "id", DataType: "uint", PrimaryKey: true},
			}),
		}

		comparer := diff.NewSchemaComparer(nil)
		schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
		require.NoError(t, err)

		assert.Empty(t, schemaDiff.TablesToCreate, "Should handle case-insensitive table names")
		assert.Empty(t, schemaDiff.TablesToDrop)
		assert.Empty(t, schemaDiff.TablesToModify)
	})
}

// Helper function to create test schemas
func createTestSchema(tableName string, fields []*schema.Field) *schema.Schema {
	return &schema.Schema{
		Name:   tableName,
		Table:  tableName,
		Fields: fields,
	}
}
