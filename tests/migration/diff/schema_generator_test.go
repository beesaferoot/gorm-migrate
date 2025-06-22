package migration

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beesaferoot/gorm-schema/migration/diff"
)

type TestModel struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"column:name;not null"`
	Email string `gorm:"column:email;unique"`
	Age   int    `gorm:"column:age"`
}

type TestModelWithIgnoredField struct {
	ID      uint   `gorm:"primaryKey"`
	Name    string `gorm:"column:name"`
	Ignored string `gorm:"-"`
	Age     int    `gorm:"column:age"`
}

func TestGenerateMigration_ValidStruct(t *testing.T) {
	modelType := reflect.TypeOf(TestModel{})
	migration, err := diff.GenerateMigration(modelType, "test_migration")
	require.NoError(t, err)

	assert.Contains(t, migration, "package migrations")
	assert.Contains(t, migration, "import \"gorm.io/gorm\"")
	assert.Contains(t, migration, "func Migrate(db *gorm.DB) error")
	assert.Contains(t, migration, "CREATE TABLE testmodel")
	assert.Contains(t, migration, "DROP TABLE testmodel")
}

func TestGenerateMigration_InvalidType(t *testing.T) {
	modelType := reflect.TypeOf("string")
	_, err := diff.GenerateMigration(modelType, "test_migration")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected struct type")
}

func TestGenerateMigration_WithConstraints(t *testing.T) {
	modelType := reflect.TypeOf(TestModel{})
	migration, err := diff.GenerateMigration(modelType, "test_migration")
	require.NoError(t, err)

	assert.Contains(t, migration, "id INTEGER PRIMARY KEY")
	assert.Contains(t, migration, "name VARCHAR(255) NOT NULL")
	assert.Contains(t, migration, "email VARCHAR(255) UNIQUE")
	assert.Contains(t, migration, "age INTEGER")
}

func TestGenerateMigration_WithIgnoredField(t *testing.T) {
	modelType := reflect.TypeOf(TestModelWithIgnoredField{})
	migration, err := diff.GenerateMigration(modelType, "test_migration")
	require.NoError(t, err)

	assert.Contains(t, migration, "id INTEGER PRIMARY KEY")
	assert.Contains(t, migration, "name VARCHAR(255)")
	assert.Contains(t, migration, "age INTEGER")
	assert.NotContains(t, migration, "Ignored")
}

func TestGetColumnName_WithTag(t *testing.T) {
	field := reflect.StructField{
		Name: "UserName",
		Tag:  reflect.StructTag(`gorm:"column:user_name"`),
	}

	columnName := getColumnName(field)
	assert.Equal(t, "user_name", columnName)
}

func TestGetColumnName_WithoutTag(t *testing.T) {
	field := reflect.StructField{
		Name: "UserName",
		Tag:  reflect.StructTag(""),
	}

	columnName := getColumnName(field)
	assert.Equal(t, "username", columnName)
}

func TestGetSQLType_String(t *testing.T) {
	fieldType := reflect.TypeOf("")
	sqlType := getSQLType(fieldType)
	assert.Equal(t, "VARCHAR(255)", sqlType)
}

func TestGetSQLType_Int(t *testing.T) {
	fieldType := reflect.TypeOf(0)
	sqlType := getSQLType(fieldType)
	assert.Equal(t, "INTEGER", sqlType)
}

func TestGetSQLType_Int64(t *testing.T) {
	fieldType := reflect.TypeOf(int64(0))
	sqlType := getSQLType(fieldType)
	assert.Equal(t, "BIGINT", sqlType)
}

func TestGetSQLType_Bool(t *testing.T) {
	fieldType := reflect.TypeOf(false)
	sqlType := getSQLType(fieldType)
	assert.Equal(t, "BOOLEAN", sqlType)
}

func TestGetSQLType_Pointer(t *testing.T) {
	fieldType := reflect.TypeOf((*string)(nil))
	sqlType := getSQLType(fieldType)
	assert.Equal(t, "VARCHAR(255)", sqlType)
}

func getColumnName(field reflect.StructField) string {
	tag := field.Tag.Get("gorm")
	if tag == "" {
		return strings.ToLower(field.Name)
	}

	parts := strings.Split(tag, ";")
	for _, part := range parts {
		if strings.HasPrefix(part, "column:") {
			return strings.TrimPrefix(part, "column:")
		}
	}

	return strings.ToLower(field.Name)
}

func getSQLType(fieldType reflect.Type) string {
	switch fieldType.Kind() {
	case reflect.String:
		return "VARCHAR(255)"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "INTEGER"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INTEGER"
	case reflect.Uint64:
		return "BIGINT"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		return "DOUBLE PRECISION"
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Struct:
		if fieldType.Name() == "Time" {
			return "TIMESTAMP"
		}
		return ""
	case reflect.Ptr:
		return getSQLType(fieldType.Elem())
	default:
		return ""
	}
}
