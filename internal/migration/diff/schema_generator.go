package diff

import (
	"fmt"
	"reflect"
	"strings"
)

// GenerateMigration generates a migration file for a GORM model
func GenerateMigration(modelType reflect.Type, name string) (string, error) {
	if modelType.Kind() != reflect.Struct {
		return "", fmt.Errorf("expected struct type, got %s", modelType.Kind())
	}

	// Generate up migration
	var upSQL strings.Builder
	upSQL.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", strings.ToLower(modelType.Name())))

	// Process each field
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		tag := field.Tag.Get("gorm")

		// Skip if field is not meant to be in the database
		if strings.Contains(tag, "-") {
			continue
		}

		// Get column name from tag or use field name
		columnName := getColumnName(field)
		if columnName == "" {
			continue
		}

		// Get SQL type from field type
		sqlType := getSQLType(field.Type)
		if sqlType == "" {
			continue
		}

		// Add column definition
		upSQL.WriteString(fmt.Sprintf("  %s %s", columnName, sqlType))

		// Add constraints
		if strings.Contains(tag, "primaryKey") {
			upSQL.WriteString(" PRIMARY KEY")
		}
		if strings.Contains(tag, "not null") {
			upSQL.WriteString(" NOT NULL")
		}
		if strings.Contains(tag, "unique") {
			upSQL.WriteString(" UNIQUE")
		}

		// Add comma if not last field
		if i < modelType.NumField()-1 {
			upSQL.WriteString(",")
		}
		upSQL.WriteString("\n")
	}

	upSQL.WriteString(");\n")

	// Generate down migration
	downSQL := fmt.Sprintf("DROP TABLE %s;\n", strings.ToLower(modelType.Name()))

	// Generate migration file content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("package migrations\n\n"))
	content.WriteString("import \"gorm.io/gorm\"\n\n")
	content.WriteString(fmt.Sprintf("func Migrate(db *gorm.DB) error {\n"))
	content.WriteString("\t// Up migration\n")
	content.WriteString(fmt.Sprintf("\tif err := db.Exec(`%s`).Error; err != nil {\n", upSQL.String()))
	content.WriteString("\t\treturn err\n")
	content.WriteString("\t}\n\n")
	content.WriteString("\t// Down migration\n")
	content.WriteString(fmt.Sprintf("\tif err := db.Exec(`%s`).Error; err != nil {\n", downSQL))
	content.WriteString("\t\treturn err\n")
	content.WriteString("\t}\n\n")
	content.WriteString("\treturn nil\n")
	content.WriteString("}\n")

	return content.String(), nil
}

// getColumnName extracts the column name from a field's GORM tag or uses the field name
func getColumnName(field reflect.StructField) string {
	tag := field.Tag.Get("gorm")
	if tag == "" {
		return strings.ToLower(field.Name)
	}

	// Look for column name in tag
	parts := strings.Split(tag, ";")
	for _, part := range parts {
		if strings.HasPrefix(part, "column:") {
			return strings.TrimPrefix(part, "column:")
		}
	}

	return strings.ToLower(field.Name)
}

// getSQLType converts a Go type to its SQL equivalent
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
