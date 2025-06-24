package diff

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Add a package-level debug flag
var debugDiffOutput = false // Set to true for detailed debug

// GetCurrentSchema gets the current database schema
func (c *SchemaComparer) GetCurrentSchema() (map[string]*schema.Schema, error) {
	return c.getCurrentSchema()
}

// GetModelSchemas gets the schema from the provided models
func (c *SchemaComparer) GetModelSchemas(models ...interface{}) (map[string]*schema.Schema, error) {
	modelSchemas := make(map[string]*schema.Schema)
	for _, model := range models {
		stmt := &gorm.Statement{DB: c.db}
		if err := stmt.Parse(model); err != nil {
			return nil, err
		}
		s := stmt.Schema
		columns := make([]*schema.Field, 0)
		seenColumns := make(map[string]bool)
		for _, field := range s.Fields {
			if field.DBName == "" {
				continue
			}
			if seenColumns[field.DBName] {
				continue
			}
			seenColumns[field.DBName] = true
			columns = append(columns, field)
		}
		// If the model embeds gorm.Model, ensure default columns are present
		if embedsGormModel(reflect.TypeOf(model)) {
			for _, def := range gormDefaultFields() {
				if !seenColumns[def.DBName] {
					// Mark default GORM fields to be ignored in migrations
					def.IgnoreMigration = true
					columns = append(columns, def)
					seenColumns[def.DBName] = true
				}
			}
		}
		copySchema := schema.Schema{
			Name:          s.Name,
			Table:         s.Table,
			Fields:        columns,
			Relationships: schema.Relationships{},
		}
		modelSchemas[s.Table] = &copySchema
	}

	// Second pass: iterate over relationships and set up foreign keys properly
	for _, s := range modelSchemas {
		relationships := schema.Relationships{}

		for _, rel := range s.Relationships.BelongsTo {
			if rel.Field != nil {
				// Find the actual foreign key field by looking at the GORM tag
				var fkField *schema.Field
				if rel.Field.TagSettings != nil {
					// Get the foreign key field name from the GORM tag
					if fkFieldName, exists := rel.Field.TagSettings["FOREIGNKEY"]; exists {
						// Look for the field with this name
						for _, field := range s.Fields {
							if field.DBName == strings.ToLower(fkFieldName) {
								fkField = field
								break
							}
						}
					}
				}

				// Fallback: try to find by naming convention if tag lookup failed
				if fkField == nil {
					for _, field := range s.Fields {
						// Look for the foreign key field that matches the relationship
						if rel.FieldSchema != nil {
							// Check if this field is the foreign key for the relationship
							expectedFKName := strings.ToLower(rel.FieldSchema.Name) + "_id"
							expectedFKName2 := strings.ToLower(rel.FieldSchema.Name) + "id"
							if field.DBName == expectedFKName || field.DBName == expectedFKName2 {
								fkField = field
								break
							}
						}
					}
				}

				// If we found the foreign key field, create the relationship
				if fkField != nil && rel.FieldSchema != nil {
					// Find the referenced schema by name
					var referencedSchema *schema.Schema
					for _, relatedSchema := range modelSchemas {
						if relatedSchema.Name == rel.FieldSchema.Name {
							referencedSchema = relatedSchema
							break
						}
					}

					if referencedSchema != nil {
						// Create a new relationship with the correct foreign key field and referenced schema
						newRel := &schema.Relationship{
							Type:        schema.BelongsTo,
							Field:       fkField,
							Schema:      referencedSchema, // This should be the parent table
							FieldSchema: rel.FieldSchema,
						}
						relationships.BelongsTo = append(relationships.BelongsTo, newRel)
					}
				}
			}
		}
		for _, rel := range s.Relationships.HasMany {
			if rel.Field != nil {
				relationships.HasMany = append(relationships.HasMany, rel)
			}
		}
		for _, rel := range s.Relationships.HasOne {
			if rel.Field != nil {
				relationships.HasOne = append(relationships.HasOne, rel)
			}
		}
		for _, rel := range s.Relationships.Many2Many {
			if rel.Field != nil {
				relationships.Many2Many = append(relationships.Many2Many, rel)
			}
		}
		// Create a new relationships instance to avoid copying locks
		var newRelationships schema.Relationships
		newRelationships.BelongsTo = append(newRelationships.BelongsTo, relationships.BelongsTo...)
		newRelationships.HasMany = append(newRelationships.HasMany, relationships.HasMany...)
		newRelationships.HasOne = append(newRelationships.HasOne, relationships.HasOne...)
		newRelationships.Many2Many = append(newRelationships.Many2Many, relationships.Many2Many...)
		// Use pointer indirection to update the fields directly, not the struct
		s.Relationships.BelongsTo = newRelationships.BelongsTo
		s.Relationships.HasMany = newRelationships.HasMany
		s.Relationships.HasOne = newRelationships.HasOne
		s.Relationships.Many2Many = newRelationships.Many2Many
	}

	// Sort tables based on dependencies
	sortedTables := make([]*schema.Schema, 0, len(modelSchemas))
	visited := make(map[string]bool)
	var visit func(*schema.Schema)
	visit = func(s *schema.Schema) {
		if visited[s.Name] {
			return
		}
		visited[s.Name] = true
		// Visit all referenced tables first
		for _, rel := range s.Relationships.BelongsTo {
			if rel.Schema != nil {
				for _, relatedSchema := range modelSchemas {
					if relatedSchema.Name == rel.Schema.Name {
						visit(relatedSchema)
						break
					}
				}
			}
		}
		sortedTables = append(sortedTables, s)
	}

	// Visit all schemas
	for _, s := range modelSchemas {
		visit(s)
	}

	// Convert sorted schemas back to a map
	sortedModelSchemas := make(map[string]*schema.Schema)
	for _, s := range sortedTables {
		sortedModelSchemas[s.Name] = s
	}

	return sortedModelSchemas, nil
}

// CompareSchemas compares two schemas and returns the differences
func (c *SchemaComparer) CompareSchemas(current, target map[string]*schema.Schema) (*SchemaDiff, error) {
	return c.compareSchemas(current, target)
}

// getCurrentSchema retrieves the current database schema using GORM's Migrator
func (c *SchemaComparer) getCurrentSchema() (map[string]*schema.Schema, error) {
	db, ok := any(c.db).(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("invalid db instance")
	}

	migrator := db.Migrator()

	tables, err := migrator.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %v", err)
	}

	schemas := make(map[string]*schema.Schema)

	for _, tableName := range tables {
		// Skip migration tracking tables
		if tableName == "migration_records" || tableName == "schema_migrations" {
			continue
		}

		columns, err := migrator.ColumnTypes(tableName)
		if err != nil {
			continue
		}

		var fields []*schema.Field
		for _, col := range columns {
			isPrimaryKey, _ := col.PrimaryKey()
			isAutoIncrement, _ := col.AutoIncrement()
			defaultValue, _ := col.DefaultValue()
			length, _ := col.Length()
			precision, scale, _ := col.DecimalSize()
			nullable, _ := col.Nullable()

			field := &schema.Field{
				Name:          toExportedFieldName(col.Name()),
				DBName:        col.Name(),
				DataType:      schema.DataType(col.DatabaseTypeName()),
				NotNull:       !nullable,
				PrimaryKey:    isPrimaryKey,
				AutoIncrement: isAutoIncrement,
				DefaultValue:  defaultValue,
				Size:          int(length),
				Precision:     int(precision),
				Scale:         int(scale),
				Creatable:     true,
				Updatable:     true,
				Readable:      true,
			}
			fields = append(fields, field)
		}

		parsedSchema := &schema.Schema{
			Name:          toExportedFieldName(tableName),
			Table:         tableName,
			Fields:        fields,
			Relationships: schema.Relationships{},
		}

		schemas[tableName] = parsedSchema
	}

	return schemas, nil
}

// normalizeTableName converts a table name to lowercase for case-insensitive comparison
func normalizeTableName(name string) string {
	return strings.ToLower(name)
}

// compareSchemas compares two schemas and returns the differences
func (c *SchemaComparer) compareSchemas(current, target map[string]*schema.Schema) (*SchemaDiff, error) {
	diff := &SchemaDiff{
		TablesToCreate: make([]TableDiff, 0),
		TablesToDrop:   make([]string, 0),
		TablesToModify: make([]TableDiff, 0),
		TablesToRename: make([]TableRename, 0),
	}

	// Create normalized maps for case-insensitive comparison
	normalizedCurrent := make(map[string]*schema.Schema)
	normalizedTarget := make(map[string]*schema.Schema)

	for name, schema := range current {
		normalizedCurrent[normalizeTableName(name)] = schema
	}
	for name, schema := range target {
		normalizedTarget[normalizeTableName(name)] = schema
	}

	// Find tables to create and modify
	for normalizedName, targetSchema := range normalizedTarget {
		currentSchema, exists := normalizedCurrent[normalizedName]
		if !exists {
			// Table needs to be created
			// Use compareTable with an empty current schema to ensure relationships are processed
			emptySchema := &schema.Schema{
				Table:         targetSchema.Table,
				Fields:        []*schema.Field{},
				Relationships: schema.Relationships{},
			}
			tableDiff := c.compareTable(emptySchema, targetSchema)
			diff.TablesToCreate = append(diff.TablesToCreate, tableDiff)
		} else {
			// Table exists, check for modifications
			tableDiff := c.compareTable(currentSchema, targetSchema)
			if !tableDiff.IsEmpty() {
				diff.TablesToModify = append(diff.TablesToModify, tableDiff)
			}
		}
	}

	// Find tables to drop
	for normalizedName := range normalizedCurrent {
		if _, exists := normalizedTarget[normalizedName]; !exists {
			// Find the original table name to add to TablesToDrop
			for originalName := range current {
				if normalizeTableName(originalName) == normalizedName {
					diff.TablesToDrop = append(diff.TablesToDrop, originalName)
					break
				}
			}
		}
	}

	return diff, nil
}

// CompareTable compares two table schemas and returns a TableDiff using GORM types
func (c *SchemaComparer) CompareTable(current, target *schema.Schema) TableDiff {
	return c.compareTable(current, target)
}

// compareTable compares two table schemas and returns a TableDiff using GORM types
func (c *SchemaComparer) compareTable(current, target *schema.Schema) TableDiff {
	diff := TableDiff{
		Schema:            target,
		FieldsToAdd:       make([]*schema.Field, 0),
		FieldsToDrop:      make([]*schema.Field, 0),
		FieldsToModify:    make([]*schema.Field, 0),
		FieldsToRename:    make([]ColumnRename, 0),
		IndexesToAdd:      make([]*schema.Index, 0),
		IndexesToDrop:     make([]*schema.Index, 0),
		IndexesToModify:   make([]*schema.Index, 0),
		ForeignKeysToAdd:  make([]*schema.Relationship, 0),
		ForeignKeysToDrop: make([]*schema.Relationship, 0),
	}

	currentFields := make(map[string]*schema.Field)
	for _, field := range current.Fields {
		currentFields[field.DBName] = normalizeFieldMetadata(field)
	}

	targetFields := make(map[string]*schema.Field)
	for _, field := range target.Fields {
		targetFields[field.DBName] = normalizeFieldMetadata(field)
	}

	for _, field := range gormDefaultFields() {
		if _, exists := targetFields[normalizeFieldName(field.DBName)]; exists {
			targetFields[normalizeFieldName(field.DBName)].IgnoreMigration = true
		}
		if _, exists := currentFields[normalizeFieldName(field.DBName)]; exists {
			currentFields[normalizeFieldName(field.DBName)].IgnoreMigration = true
		}
	}

	for normName, targetField := range targetFields {
		if targetField == nil || targetField.IgnoreMigration || targetField.DBName == "" {
			continue
		}
		if currentField, exists := currentFields[normName]; !exists {
			if debugDiffOutput {
				fmt.Printf("[DEBUG] targetField: %+v\n", targetField.Name)
				fmt.Printf("[DEBUG] Field addition detected for %s.%s\n\n", target.Table, targetField.DBName)
			}
			diff.FieldsToAdd = append(diff.FieldsToAdd, targetField)
		} else if !fieldsEqual(currentField, targetField) {
			if debugDiffOutput {
				fmt.Printf("[DEBUG] currentField: %+v\n", currentField.Name)
				fmt.Printf("[DEBUG] Field modification detected for %s.%s: current type=%v, target type=%v\n\n", target.Table, targetField.DBName, currentField.DataType, targetField.DataType)
			}
			diff.FieldsToModify = append(diff.FieldsToModify, targetField)
		}
	}
	for normName, currentField := range currentFields {
		if currentField.IgnoreMigration {
			continue
		}
		if _, exists := targetFields[normName]; !exists {
			if debugDiffOutput {
				fmt.Printf("[DEBUG] currentField: %+v\n", currentField.Name)
				fmt.Printf("[DEBUG] Field drop detected for %s.%s\n\n", current.Table, currentField.DBName)
			}
			diff.FieldsToDrop = append(diff.FieldsToDrop, currentField)
		}
	}

	// Only include index diffs for new tables (when current.Fields is empty)
	if len(current.Fields) == 0 {
		currentIndexes := make(map[string]*schema.Index)
		for _, idx := range current.ParseIndexes() {
			currentIndexes[idx.Name] = idx
		}
		targetIndexes := make(map[string]*schema.Index)
		for _, idx := range target.ParseIndexes() {
			targetIndexes[idx.Name] = idx
		}
		for name, targetIdx := range targetIndexes {
			if currentIdx, exists := currentIndexes[name]; !exists {
				diff.IndexesToAdd = append(diff.IndexesToAdd, targetIdx)
			} else if !indexesEqual(currentIdx, targetIdx) {
				diff.IndexesToModify = append(diff.IndexesToModify, targetIdx)
			}
		}
		for name, currentIdx := range currentIndexes {
			if _, exists := targetIndexes[name]; !exists {
				diff.IndexesToDrop = append(diff.IndexesToDrop, currentIdx)
			}
		}
	}

	// Foreign key diffs are still ignored for now
	return diff
}

// normalizeFieldMetadata normalizes field metadata for comparison, ignoring GORM-specific metadata that doesn't affect DB schema
func normalizeFieldMetadata(field *schema.Field) *schema.Field {
	if field == nil {
		return nil
	}

	// Create a copy of the field with normalized metadata
	normalized := &schema.Field{
		Name:            field.Name,
		DBName:          field.DBName,
		DataType:        field.DataType,
		GORMDataType:    field.GORMDataType,
		PrimaryKey:      field.PrimaryKey,
		AutoIncrement:   field.AutoIncrement,
		NotNull:         field.NotNull,
		Unique:          field.Unique,
		DefaultValue:    field.DefaultValue,
		Size:            field.Size,
		Precision:       field.Precision,
		Scale:           field.Scale,
		Comment:         field.Comment,
		IgnoreMigration: field.IgnoreMigration,
	}

	return normalized
}

// fieldsEqual compares two *schema.Field for relevant diff purposes
func fieldsEqual(a, b *schema.Field) bool {
	// Normalize field names case-insensitively
	if !strings.EqualFold(a.DBName, b.DBName) {
		return false
	}

	// Compare normalized data types
	if normalizeDBType(a.DataType) != normalizeDBType(b.DataType) {
		return false
	}

	// For primary keys and auto-increment fields, ignore nullability differences
	// (GORM often sets these differently than the database)
	if a.PrimaryKey != b.PrimaryKey {
		return false
	}

	if a.Unique != b.Unique {
		return false
	}

	// Normalize and compare default values
	if normalizeDefaultValue(a.DefaultValue) != normalizeDefaultValue(b.DefaultValue) {
		return false
	}

	// Compare auto-increment status
	if a.AutoIncrement != b.AutoIncrement {
		return false
	}

	// For non-primary keys, compare nullability
	if !a.PrimaryKey && a.NotNull != b.NotNull {
		return false
	}

	return true
}

// normalizeDBType normalizes Go/GORM/Postgres types for DB comparison
func normalizeDBType(dt schema.DataType) string {
	dtStr := strings.ToLower(string(dt))
	// Normalize integer types
	if dtStr == "int" || dtStr == "int32" || dtStr == "int4" || dtStr == "int64" || dtStr == "int8" || dtStr == "uint" || dtStr == "bigint" {
		return "bigint"
	}
	// Normalize float/decimal types
	if dtStr == "float64" || dtStr == "float32" || dtStr == "float" || dtStr == "real" || dtStr == "numeric" || dtStr == "decimal" || strings.HasPrefix(dtStr, "decimal(") || dtStr == "float8" || dtStr == "double precision" {
		return "decimal"
	}
	// Normalize string types
	if dtStr == "string" || dtStr == "varchar" || dtStr == "text" || dtStr == "character varying" {
		return "varchar"
	}
	if dtStr == "bool" || dtStr == "boolean" {
		return "boolean"
	}
	if dtStr == "time" || dtStr == "timestamp" || dtStr == "timestamp without time zone" || dtStr == "timestamp with time zone" {
		return "timestamp"
	}
	if dtStr == "json" || dtStr == "jsonb" {
		return "jsonb"
	}
	return dtStr
}

// normalizeDefaultValue normalizes default values for comparison
func normalizeDefaultValue(dv string) string {
	if dv == "" {
		return ""
	}

	// Remove quotes and normalize common defaults
	dv = strings.Trim(dv, "'\"")
	dv = strings.ToLower(dv)

	// Normalize auto-increment sequences
	if strings.HasPrefix(dv, "nextval") {
		return "auto_increment"
	}

	// Normalize common default values
	switch dv {
	case "null", "default null":
		return ""
	case "true", "false":
		return dv
	case "0", "0.0":
		return "0"
	case "now()", "current_timestamp", "current_timestamp()":
		return "current_timestamp"
	}

	return dv
}

// isRelationshipField checks if a field is a relationship field (not a DB column)
func isRelationshipField(field *schema.Field) bool {
	// Skip if FieldType is nil (database-extracted fields may not have this)
	if field.FieldType == nil {
		return false
	}

	if field.DBName == "" {
		return true
	}

	// Check if it's a struct pointer (relationship field)
	if field.FieldType.Kind() == reflect.Ptr && field.FieldType.Elem().Kind() == reflect.Struct {
		return true
	}
	// Check if it's a slice (one-to-many relationship)
	if field.FieldType.Kind() == reflect.Slice {
		return true
	}
	// Check if it has a foreign key tag but is not the actual foreign key column
	if field.Tag.Get("foreignKey") != "" && !strings.HasSuffix(field.DBName, "_id") {
		return true
	}
	// Check if it's a struct (embedded or relationship)
	if field.FieldType.Kind() == reflect.Struct {
		return true
	}
	if strings.HasPrefix(field.Tag.Get("gorm"), "foreignKey:") {
		return true
	}
	return false
}

// indexesEqual compares two schema.Index for relevant diff purposes
func indexesEqual(a, b *schema.Index) bool {
	if a.Name != b.Name || a.Option != b.Option || len(a.Fields) != len(b.Fields) {
		return false
	}
	for i := range a.Fields {
		if a.Fields[i].DBName != b.Fields[i].DBName {
			return false
		}
	}
	return true
}



// toExportedFieldName converts snake_case or lower to ExportedCamelCase
func toExportedFieldName(name string) string {
	if name == "" {
		return "Field"
	}
	// Split by _ and capitalize each part
	result := ""
	capitalizeNext := true
	for _, r := range name {
		if r == '_' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			r = rune(toUpper(byte(r)))
			capitalizeNext = false
		}
		result += string(r)
	}
	return result
}

func toUpper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 'a' + 'A'
	}
	return b
}



// embedsGormModel returns true if the type embeds gorm.Model
func embedsGormModel(t reflect.Type) bool {
	t = indirectType(t)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous && f.Type.PkgPath() == "gorm.io/gorm" && f.Type.Name() == "Model" {
			return true
		}
	}
	return false
}

func indirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// gormDefaultFields returns the default GORM fields as schema.Field
func gormDefaultFields() []*schema.Field {
	return []*schema.Field{
		{Name: "ID", DBName: "id", DataType: "uint", PrimaryKey: true, AutoIncrement: true, IgnoreMigration: true},
		{Name: "CreatedAt", DBName: "created_at", DataType: "time", IgnoreMigration: true},
		{Name: "UpdatedAt", DBName: "updated_at", DataType: "time", IgnoreMigration: true},
		{Name: "DeletedAt", DBName: "deleted_at", DataType: "time", IgnoreMigration: true},
	}
}

// normalizeFieldName: normalize field name for comparison (case-insensitive, underscores ignored)
func normalizeFieldName(name string) string {
	var result []rune
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
