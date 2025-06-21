package diff

import (
	"fmt"
	"reflect"
	"strings"
)

func GenerateMigration(modelType reflect.Type, name string) (string, error) {
	if modelType.Kind() != reflect.Struct {
		return "", fmt.Errorf("expected struct type, got %s", modelType.Kind())
	}

	var upSQL strings.Builder
	upSQL.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", strings.ToLower(modelType.Name())))

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		tag := field.Tag.Get("gorm")

		if strings.Contains(tag, "-") {
			continue
		}

		columnName := getColumnName(field)
		if columnName == "" {
			continue
		}

		sqlType := getSQLType(field.Type)
		if sqlType == "" {
			continue
		}

		upSQL.WriteString(fmt.Sprintf("  %s %s", columnName, sqlType))

		if strings.Contains(tag, "primaryKey") {
			upSQL.WriteString(" PRIMARY KEY")
		}
		if strings.Contains(tag, "not null") {
			upSQL.WriteString(" NOT NULL")
		}
		if strings.Contains(tag, "unique") {
			upSQL.WriteString(" UNIQUE")
		}

		if i < modelType.NumField()-1 {
			upSQL.WriteString(",")
		}
		upSQL.WriteString("\n")
	}

	upSQL.WriteString(");\n")

	downSQL := fmt.Sprintf("DROP TABLE %s;\n", strings.ToLower(modelType.Name()))

	var content strings.Builder
	content.WriteString(fmt.Sprintf("package migrations\n\n"))
	content.WriteString("import \"gorm.io/gorm\"\n\n")
	content.WriteString(fmt.Sprintf("func Migrate(db *gorm.DB) error {\n"))
	content.WriteString(fmt.Sprintf("\tif err := db.Exec(`%s`).Error; err != nil {\n", upSQL.String()))
	content.WriteString("\t\treturn err\n")
	content.WriteString("\t}\n\n")
	content.WriteString(fmt.Sprintf("\tif err := db.Exec(`%s`).Error; err != nil {\n", downSQL))
	content.WriteString("\t\treturn err\n")
	content.WriteString("\t}\n\n")
	content.WriteString("\treturn nil\n")
	content.WriteString("}\n")

	return content.String(), nil
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
