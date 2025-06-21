package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"

	"gorm-schema/internal/migration/diff"
)


func TestSchemaComparer_CompareSchemas_Unit(t *testing.T) {
	comparer := &diff.SchemaComparer{}

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
	comparer := &diff.SchemaComparer{}

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
	comparer := &diff.SchemaComparer{}

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
	comparer := &diff.SchemaComparer{}

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
