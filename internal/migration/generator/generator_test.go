package generator

import (
	"gorm-schema/internal/migration/diff"
	"os"
	"strings"
	"testing"

	"gorm.io/gorm/schema"

	"github.com/stretchr/testify/require"
)

func TestGenerateCreateTableSQL_ForeignKey(t *testing.T) {
	gen := NewGenerator("migrations")
	table := diff.TableDiff{
		Schema: &schema.Schema{Table: "orders"},
		FieldsToAdd: []*schema.Field{
			{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: true},
			{DBName: "user_id", DataType: "int", NotNull: true},
		},
		ForeignKeysToAdd: []*schema.Relationship{
			{
				Field:  &schema.Field{DBName: "user_id"},
				Schema: &schema.Schema{Table: "users"},
			},
		},
	}

	sql := gen.generateCreateTableSQL(table)
	require.Contains(t, sql, "CONSTRAINT fk_orders_user_id_fkey FOREIGN KEY (\"user_id\") REFERENCES \"users\"(id) ON DELETE CASCADE")
	require.Contains(t, sql, "CREATE TABLE \"orders\" (")
	require.NotContains(t, sql, "DEFAULT NULL\n\tDEFAULT NULL")
}

func TestGenerateCreateTableSQL_Indexes(t *testing.T) {
	gen := NewGenerator("migrations")
	table := diff.TableDiff{
		Schema: &schema.Schema{Table: "products"},
		FieldsToAdd: []*schema.Field{
			{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: true},
			{DBName: "sku", DataType: "string", NotNull: true},
			{DBName: "category_id", DataType: "int", NotNull: true},
		},
		IndexesToAdd: []*schema.Index{
			{
				Name:   "products_sku_unique",
				Fields: []schema.IndexOption{{Field: &schema.Field{DBName: "sku"}}},
				Option: "UNIQUE",
			},
			{
				Name:   "products_category_id_idx",
				Fields: []schema.IndexOption{{Field: &schema.Field{DBName: "category_id"}}},
				Option: "",
			},
		},
	}

	sql := gen.generateCreateTableSQL(table)
	require.Contains(t, sql, "CONSTRAINT products_sku_unique UNIQUE (\"sku\")")
	require.Contains(t, sql, "CREATE INDEX products_category_id_idx ON \"products\" (\"category_id\");")
}

func TestGenerateCreateTableSQL_Formatting(t *testing.T) {
	gen := NewGenerator("migrations")
	table := diff.TableDiff{
		Schema: &schema.Schema{Table: "test_table"},
		FieldsToAdd: []*schema.Field{
			{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: true},
			{DBName: "name", DataType: "string", NotNull: true},
		},
	}

	sql := gen.generateCreateTableSQL(table)
	require.Contains(t, sql, "CREATE TABLE \"test_table\" (")
	require.Contains(t, sql, "id SERIAL NOT NULL PRIMARY KEY")
	require.Contains(t, sql, "name varchar(255) NOT NULL")
	require.NotContains(t, sql, ",\n\n")
	require.NotContains(t, sql, ",\n);")
}

func cleanupTestMigrations(t *testing.T, pattern string) {
	files, err := os.ReadDir("migrations")
	if err != nil {
		return
	}
	for _, file := range files {
		if strings.Contains(file.Name(), pattern) {
			_ = os.Remove("migrations/" + file.Name())
		}
	}
}

func TestGenerateMigration_FullProcess(t *testing.T) {
	t.Cleanup(func() { cleanupTestMigrations(t, "test_migration") })
	gen := NewGenerator("migrations")
	schemaDiff := &diff.SchemaDiff{
		TablesToCreate: []diff.TableDiff{
			{
				Schema: &schema.Schema{Table: "users"},
				FieldsToAdd: []*schema.Field{
					{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: false},
					{DBName: "name", DataType: "string", NotNull: false},
				},
			},
			{
				Schema: &schema.Schema{Table: "orders"},
				FieldsToAdd: []*schema.Field{
					{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: false},
					{DBName: "user_id", DataType: "int", NotNull: false},
				},
				ForeignKeysToAdd: []*schema.Relationship{
					{
						Field:  &schema.Field{DBName: "user_id"},
						Schema: &schema.Schema{Table: "users"},
					},
				},
			},
		},
	}

	gen.SetSchemaDiff(schemaDiff)
	err := gen.CreateMigration("test_migration")
	require.NoError(t, err)

	// List the migrations directory to find the generated file
	files, err := os.ReadDir("migrations")
	require.NoError(t, err)
	var migrationFile string
	for _, file := range files {
		if strings.Contains(file.Name(), "test_migration") {
			migrationFile = file.Name()
			break
		}
	}
	require.NotEmpty(t, migrationFile, "Generated migration file not found")

	// Read the generated migration file
	content, err := os.ReadFile("migrations/" + migrationFile)
	require.NoError(t, err)
	sql := string(content)

	// Verify the generated SQL
	require.Contains(t, sql, "CREATE TABLE \"users\" (")
	require.Contains(t, sql, "CREATE TABLE \"orders\" (")
	require.Contains(t, sql, "CONSTRAINT fk_orders_user_id_fkey")
	require.Contains(t, sql, "FOREIGN KEY (\"user_id\")")
	require.Contains(t, sql, "REFERENCES \"users\"(id)")
	require.Contains(t, sql, "ON DELETE CASCADE")
}

func TestGenerateMigration_EmptyDiff(t *testing.T) {
	t.Cleanup(func() { cleanupTestMigrations(t, "empty_migration") })
	gen := NewGenerator("migrations")
	schemaDiff := &diff.SchemaDiff{} // Empty diff
	gen.SetSchemaDiff(schemaDiff)
	err := gen.CreateMigration("empty_migration")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no schema changes detected")
}

func TestGenerateMigration_MissingColumns(t *testing.T) {
	t.Cleanup(func() { cleanupTestMigrations(t, "missing_columns_migration") })
	gen := NewGenerator("migrations")
	schemaDiff := &diff.SchemaDiff{
		TablesToCreate: []diff.TableDiff{
			{
				Schema: &schema.Schema{Table: "test_table"},
				// No columns added
			},
		},
	}
	gen.SetSchemaDiff(schemaDiff)
	err := gen.CreateMigration("missing_columns_migration")
	require.NoError(t, err)
	// Verify the generated migration file handles missing columns gracefully
}

func TestGenerateMigration_InvalidTableName(t *testing.T) {
	t.Cleanup(func() { cleanupTestMigrations(t, "invalid_table_migration") })
	gen := NewGenerator("migrations")
	schemaDiff := &diff.SchemaDiff{
		TablesToCreate: []diff.TableDiff{
			{
				Schema: &schema.Schema{Table: ""}, // Empty table name
				FieldsToAdd: []*schema.Field{
					{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: false},
				},
			},
		},
	}
	gen.SetSchemaDiff(schemaDiff)
	err := gen.CreateMigration("invalid_table_migration")
	require.Error(t, err)
	require.Contains(t, err.Error(), "table name cannot be empty")
}

func TestGenerateMigration_InvalidColumnType(t *testing.T) {
	t.Cleanup(func() { cleanupTestMigrations(t, "invalid_column_migration") })
	gen := NewGenerator("migrations")
	schemaDiff := &diff.SchemaDiff{
		TablesToCreate: []diff.TableDiff{
			{
				Schema: &schema.Schema{Table: "test_table"},
				FieldsToAdd: []*schema.Field{
					{DBName: "id", DataType: "invalid_type", PrimaryKey: true, NotNull: false},
				},
			},
		},
	}
	gen.SetSchemaDiff(schemaDiff)
	err := gen.CreateMigration("invalid_column_migration")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported column type")
}

func TestGenerateMigration_MissingForeignKeyReference(t *testing.T) {
	t.Cleanup(func() { cleanupTestMigrations(t, "missing_fk_migration") })
	gen := NewGenerator("migrations")
	schemaDiff := &diff.SchemaDiff{
		TablesToCreate: []diff.TableDiff{
			{
				Schema: &schema.Schema{Table: "orders"},
				FieldsToAdd: []*schema.Field{
					{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: false},
					{DBName: "user_id", DataType: "int", NotNull: false},
				},
				ForeignKeysToAdd: []*schema.Relationship{
					{
						Name:   "orders_user_id_fkey",
						Field:  &schema.Field{DBName: "user_id"},
						Schema: &schema.Schema{Table: "nonexistent_table"}, // Table doesn't exist
					},
				},
			},
		},
	}
	gen.SetSchemaDiff(schemaDiff)
	err := gen.CreateMigration("missing_fk_migration")
	require.Error(t, err)
	require.Contains(t, err.Error(), "table nonexistent_table not found")
}

func TestGenerateMigration_DuplicateColumnNames(t *testing.T) {
	t.Cleanup(func() { cleanupTestMigrations(t, "duplicate_columns_migration") })
	gen := NewGenerator("migrations")
	schemaDiff := &diff.SchemaDiff{
		TablesToCreate: []diff.TableDiff{
			{
				Schema: &schema.Schema{Table: "test_table"},
				FieldsToAdd: []*schema.Field{
					{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: false},
					{DBName: "id", DataType: "int", PrimaryKey: false, NotNull: false}, // Duplicate column name
				},
			},
		},
	}
	gen.SetSchemaDiff(schemaDiff)
	err := gen.CreateMigration("duplicate_columns_migration")
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate column name")
}

func TestGenerateMigration_InvalidIndexDefinition(t *testing.T) {
	t.Cleanup(func() { cleanupTestMigrations(t, "invalid_index_migration") })
	gen := NewGenerator("migrations")
	schemaDiff := &diff.SchemaDiff{
		TablesToCreate: []diff.TableDiff{
			{
				Schema: &schema.Schema{Table: "test_table"},
				FieldsToAdd: []*schema.Field{
					{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: false},
					{DBName: "name", DataType: "string", NotNull: false},
				},
				IndexesToAdd: []*schema.Index{
					{
						Name:   "invalid_index",
						Fields: []schema.IndexOption{{Field: &schema.Field{DBName: "nonexistent_column"}}}, // Column doesn't exist
						Option: "UNIQUE",
					},
				},
			},
		},
	}
	gen.SetSchemaDiff(schemaDiff)
	err := gen.CreateMigration("invalid_index_migration")
	require.Error(t, err)
	require.Contains(t, err.Error(), "index references non-existent column")
}

func TestGenerateMigration_ComplexRelationships(t *testing.T) {
	t.Cleanup(func() { cleanupTestMigrations(t, "complex_relationships_migration") })
	gen := NewGenerator("migrations")
	schemaDiff := &diff.SchemaDiff{
		TablesToCreate: []diff.TableDiff{
			{
				Schema: &schema.Schema{Table: "users"},
				FieldsToAdd: []*schema.Field{
					{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: false},
					{DBName: "name", DataType: "string", NotNull: false},
				},
			},
			{
				Schema: &schema.Schema{Table: "orders"},
				FieldsToAdd: []*schema.Field{
					{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: false},
					{DBName: "user_id", DataType: "int", NotNull: false},
				},
				ForeignKeysToAdd: []*schema.Relationship{
					{
						Name:   "orders_user_id_fkey",
						Field:  &schema.Field{DBName: "user_id"},
						Schema: &schema.Schema{Table: "users"},
					},
				},
			},
			{
				Schema: &schema.Schema{Table: "order_items"},
				FieldsToAdd: []*schema.Field{
					{DBName: "id", DataType: "int", PrimaryKey: true, NotNull: false},
					{DBName: "order_id", DataType: "int", NotNull: false},
					{DBName: "product_id", DataType: "int", NotNull: false},
				},
				ForeignKeysToAdd: []*schema.Relationship{
					{
						Name:   "order_items_order_id_fkey",
						Field:  &schema.Field{DBName: "order_id"},
						Schema: &schema.Schema{Table: "orders"},
					},
				},
			},
		},
	}
	gen.SetSchemaDiff(schemaDiff)
	err := gen.CreateMigration("complex_relationships_migration")
	require.NoError(t, err)

	// List the migrations directory to find the generated file
	files, err := os.ReadDir("migrations")
	require.NoError(t, err)
	var migrationFile string
	for _, file := range files {
		if strings.Contains(file.Name(), "complex_relationships_migration") {
			migrationFile = file.Name()
			break
		}
	}
	require.NotEmpty(t, migrationFile, "Generated migration file not found")

	// Read the generated migration file
	content, err := os.ReadFile("migrations/" + migrationFile)
	require.NoError(t, err)
	sql := string(content)

	// Verify the generated SQL
	require.Contains(t, sql, "CREATE TABLE \"users\" (")
	require.Contains(t, sql, "CREATE TABLE \"orders\" (")
	require.Contains(t, sql, "CREATE TABLE \"order_items\" (")
	require.Contains(t, sql, "CONSTRAINT fk_orders_user_id_fkey")
	require.Contains(t, sql, "FOREIGN KEY (\"user_id\")")
	require.Contains(t, sql, "REFERENCES \"users\"(id)")
	require.Contains(t, sql, "CONSTRAINT fk_order_items_order_id_fkey")
	require.Contains(t, sql, "FOREIGN KEY (\"order_id\")")
	require.Contains(t, sql, "REFERENCES \"orders\"(id)")
	require.Contains(t, sql, "ON DELETE CASCADE")
}
