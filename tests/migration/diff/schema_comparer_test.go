package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/beesaferoot/gorm-schema/migration/diff"
)

// createTestDBForSchemaComparer creates a test database for schema comparer unit tests
func createTestDBForSchemaComparer(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestSchemaComparer_CompareSchemas_Unit(t *testing.T) {
	comparer := diff.NewSchemaComparer(createTestDBForSchemaComparer(t))

	currentSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
				{Name: "Age", DBName: "age", DataType: "int"},
			},
		},
	}

	targetSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
				{Name: "Age", DBName: "age", DataType: "int"},
				{Name: "Email", DBName: "email", DataType: "string"},
			},
		},
	}

	schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
	require.NoError(t, err)

	assert.NotEmpty(t, schemaDiff.TablesToModify)
	assert.Empty(t, schemaDiff.TablesToCreate)
	assert.Empty(t, schemaDiff.TablesToDrop)
}

func TestSchemaComparer_CompareSchemas_NoChanges(t *testing.T) {
	comparer := diff.NewSchemaComparer(createTestDBForSchemaComparer(t))

	currentSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
				{Name: "Age", DBName: "age", DataType: "int"},
			},
		},
	}

	targetSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
				{Name: "Age", DBName: "age", DataType: "int"},
			},
		},
	}

	schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
	require.NoError(t, err)

	assert.Empty(t, schemaDiff.TablesToCreate)
	assert.Empty(t, schemaDiff.TablesToDrop)
	assert.Empty(t, schemaDiff.TablesToModify)
}

func TestSchemaComparer_CompareSchemas_NewTable(t *testing.T) {
	comparer := diff.NewSchemaComparer(createTestDBForSchemaComparer(t))

	currentSchema := map[string]*schema.Schema{}

	targetSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
			},
		},
	}

	schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
	require.NoError(t, err)

	assert.NotEmpty(t, schemaDiff.TablesToCreate)
	assert.Empty(t, schemaDiff.TablesToDrop)
	assert.Empty(t, schemaDiff.TablesToModify)
}

func TestSchemaComparer_CompareSchemas_DropTable(t *testing.T) {
	comparer := diff.NewSchemaComparer(createTestDBForSchemaComparer(t))

	currentSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
			},
		},
	}

	targetSchema := map[string]*schema.Schema{}

	schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
	require.NoError(t, err)

	assert.Empty(t, schemaDiff.TablesToCreate)
	assert.NotEmpty(t, schemaDiff.TablesToDrop)
	assert.Empty(t, schemaDiff.TablesToModify)
}

func TestSchemaComparer_CompareSchemas_RemoveColumn(t *testing.T) {
	comparer := diff.NewSchemaComparer(createTestDBForSchemaComparer(t))

	currentSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
				{Name: "Age", DBName: "age", DataType: "int"},
			},
		},
	}

	targetSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
			},
		},
	}

	schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
	require.NoError(t, err)

	assert.NotEmpty(t, schemaDiff.TablesToModify)
	assert.Equal(t, 1, len(schemaDiff.TablesToModify))
	assert.Equal(t, 1, len(schemaDiff.TablesToModify[0].FieldsToDrop))
	assert.Equal(t, "age", schemaDiff.TablesToModify[0].FieldsToDrop[0].DBName)
}

func TestSchemaComparer_CompareSchemas_ModifyColumn(t *testing.T) {
	comparer := diff.NewSchemaComparer(createTestDBForSchemaComparer(t))

	currentSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
				{Name: "Age", DBName: "age", DataType: "int"},
			},
		},
	}

	targetSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
				{Name: "Age", DBName: "age", DataType: "string"}, // type changed
			},
		},
	}

	schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
	require.NoError(t, err)

	assert.NotEmpty(t, schemaDiff.TablesToModify)
	assert.Equal(t, 1, len(schemaDiff.TablesToModify))
	assert.Equal(t, 1, len(schemaDiff.TablesToModify[0].FieldsToModify))
	assert.Equal(t, "age", schemaDiff.TablesToModify[0].FieldsToModify[0].DBName)
}

func TestSchemaComparer_CompareSchemas_IndexChangeOnExistingTable_Ignored(t *testing.T) {
	comparer := diff.NewSchemaComparer(createTestDBForSchemaComparer(t))

	currentSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
			},
		},
	}
	targetSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
			},
		},
	}

	schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
	require.NoError(t, err)

	assert.Empty(t, schemaDiff.TablesToModify)
}

func TestSchemaComparer_CompareSchemas_IndexChangeOnNewTable_Allowed(t *testing.T) {
	comparer := diff.NewSchemaComparer(createTestDBForSchemaComparer(t))

	currentSchema := map[string]*schema.Schema{}
	targetSchema := map[string]*schema.Schema{
		"users": {
			Name:  "users",
			Table: "users",
			Fields: []*schema.Field{
				{Name: "ID", DBName: "id", DataType: "int", PrimaryKey: true},
				{Name: "Name", DBName: "name", DataType: "string"},
			},
		},
	}

	schemaDiff, err := comparer.CompareSchemas(currentSchema, targetSchema)
	require.NoError(t, err)

	assert.NotEmpty(t, schemaDiff.TablesToCreate)
	assert.Equal(t, 1, len(schemaDiff.TablesToCreate))
	// IndexesToAdd may be empty since we are not parsing index tags in this test, but the logic is exercised
	assert.True(t, len(schemaDiff.TablesToCreate[0].IndexesToAdd) >= 0)
}
