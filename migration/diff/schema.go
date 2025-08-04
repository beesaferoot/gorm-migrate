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
	originalRelationships := make(map[string]*schema.Relationships)

	// First pass: create schemas with fields and store original relationships
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
			// Skip relationship fields that don't have DB columns
			if isRelationshipField(field) {
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

		// Store original relationships for later processing
		originalRelationships[s.Table] = &s.Relationships

		// Create a copy of the schema with fields and empty relationships
		copySchema := schema.Schema{
			Name:          s.Name,
			Table:         s.Table,
			Fields:        columns,
			Relationships: schema.Relationships{}, // Create empty relationships to avoid copying locks
		}
		modelSchemas[s.Table] = &copySchema
	}

	// Second pass: process relationships and set up foreign keys properly
	for tableName, s := range modelSchemas {
		relationships := schema.Relationships{}
		originalRel := originalRelationships[tableName]

		for _, rel := range originalRel.BelongsTo {
			// Build a map of DB columns for this schema
			dbColumns := make(map[string]*schema.Field)
			for _, field := range s.Fields {
				dbColumns[strings.ToLower(field.DBName)] = field
			}

			// Build candidate foreign key names
			var candidates []string
			if rel.Field != nil && rel.Field.TagSettings != nil {
				if fkFieldName, exists := rel.Field.TagSettings["FOREIGNKEY"]; exists {
					candidates = append(candidates, fkFieldName)
					candidates = append(candidates, normalizeFieldName(fkFieldName))
					candidates = append(candidates, strings.ToLower(fkFieldName))
				}
			}
			if rel.FieldSchema != nil {
				candidates = append(candidates, rel.FieldSchema.Name+"ID")
				candidates = append(candidates, normalizeFieldName(rel.FieldSchema.Name)+"_id")
				candidates = append(candidates, strings.ToLower(rel.FieldSchema.Name)+"id")
			}

			// Try to find a matching DB column
			var fkField *schema.Field
			for _, candidate := range candidates {
				if field, ok := dbColumns[strings.ToLower(candidate)]; ok {
					fkField = field
					break
				}
			}

			if fkField != nil && fkField.DBName != "" && rel.FieldSchema != nil {
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
						Schema:      referencedSchema,
						FieldSchema: rel.FieldSchema,
					}
					relationships.BelongsTo = append(relationships.BelongsTo, newRel)
				}
			} else if debugDiffOutput {
				fmt.Printf("[DEBUG] No valid foreign key found for relationship %s in table %s\n", rel.Name, s.Table)
			}
		}

		for _, rel := range originalRel.HasMany {
			if rel.Field != nil {
				relationships.HasMany = append(relationships.HasMany, rel)
			}
		}

		for _, rel := range originalRel.HasOne {
			if rel.Field != nil {
				relationships.HasOne = append(relationships.HasOne, rel)
			}
		}

		for _, rel := range originalRel.Many2Many {
			if rel.Field != nil {
				relationships.Many2Many = append(relationships.Many2Many, rel)
			}
		}

		// Update the schema with processed relationships
		s.Relationships.BelongsTo = relationships.BelongsTo
		s.Relationships.HasMany = relationships.HasMany
		s.Relationships.HasOne = relationships.HasOne
		s.Relationships.Many2Many = relationships.Many2Many
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

	migrator := NewSchemaMigrator(db)

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

		// Get indexes from the database
		indexes, err := migrator.GetIndexes(tableName)
		if err != nil {
			// Log error but continue - indexes are not critical for basic schema comparison
			if debugDiffOutput {
				fmt.Printf("[DEBUG] Failed to get indexes for table %s: %v\n", tableName, err)
			}
		}

		// Get relationships from the database
		relationships, err := migrator.GetRelationships(tableName)
		if err != nil {
			// Log error but continue - relationships are not critical for basic schema comparison
			if debugDiffOutput {
				fmt.Printf("[DEBUG] Failed to get relationships for table %s: %v\n", tableName, err)
			}
		}

		// Create relationships structure
		relationshipsStruct := schema.Relationships{}
		relationshipsStruct.BelongsTo = append(relationshipsStruct.BelongsTo, relationships...)

		parsedSchema := &schema.Schema{
			Name:          toExportedFieldName(tableName),
			Table:         tableName,
			Fields:        fields,
			Relationships: relationshipsStruct,
		}

		// Store database indexes and relationships for comparison
		// We'll use a custom approach to compare these later
		if len(indexes) > 0 || len(relationships) > 0 {
			// For now, we'll store this information in the schema's comment field as a marker
			// In a more sophisticated implementation, we could extend the schema structure
			if debugDiffOutput {
				fmt.Printf("[DEBUG] Found %d indexes and %d relationships for table %s\n", len(indexes), len(relationships), tableName)
			}
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
		if targetField == nil || targetField.DBName == "" {
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
		if _, exists := targetFields[normName]; !exists {
			if debugDiffOutput {
				fmt.Printf("[DEBUG] currentField: %+v\n", currentField.Name)
				fmt.Printf("[DEBUG] Field drop detected for %s.%s\n\n", current.Table, currentField.DBName)
			}
			diff.FieldsToDrop = append(diff.FieldsToDrop, currentField)
		}
	}

	currentIndexes := make(map[string]*schema.Index)
	for _, idx := range current.ParseIndexes() {
		currentIndexes[idx.Name] = idx
	}
	targetIndexes := make(map[string]*schema.Index)
	for _, idx := range target.ParseIndexes() {
		targetIndexes[idx.Name] = idx
	}

	for name, targetIdx := range targetIndexes {
		if _, exists := currentIndexes[name]; !exists {
			diff.IndexesToAdd = append(diff.IndexesToAdd, targetIdx)
		} else if !indexesEqual(currentIndexes[name], targetIdx) {
			diff.IndexesToModify = append(diff.IndexesToModify, targetIdx)
		}
	}

	if len(current.Fields) > 0 {
		for name, currentIdx := range currentIndexes {
			if _, exists := targetIndexes[name]; !exists {
				diff.IndexesToDrop = append(diff.IndexesToDrop, currentIdx)
			}
		}
	}

	currentRelationships := make(map[string]*schema.Relationship)
	for _, rel := range current.Relationships.BelongsTo {
		if rel.Field != nil {
			column_rel_ident := fmt.Sprintf("%s_%s", rel.Field.Schema.Table, rel.References[0].ForeignKey.DBName)
			currentRelationships[column_rel_ident] = rel
		}
	}

	targetRelationships := make(map[string]*schema.Relationship)
	for _, rel := range target.Relationships.BelongsTo {
		if rel.Field != nil && len(rel.References) > 0 {
			column_rel_ident := fmt.Sprintf("%s_%s", rel.Field.Schema.Table, rel.References[0].ForeignKey.DBName)
			targetRelationships[column_rel_ident] = rel
		}
	}

	for fieldName, targetRel := range targetRelationships {
		if _, exists := currentRelationships[fieldName]; !exists {
			fmt.Printf("column name %s fk does not exist\n", fieldName)
			diff.ForeignKeysToAdd = append(diff.ForeignKeysToAdd, targetRel)
		} else if !relationshipsEqual(currentRelationships[fieldName], targetRel) {
			fmt.Printf("column name %s fk rel not equal to target\n", fieldName)
			diff.ForeignKeysToAdd = append(diff.ForeignKeysToAdd, targetRel)
		}
	}

	if len(current.Fields) > 0 {
		for fieldName, currentRel := range currentRelationships {
			if _, exists := targetRelationships[fieldName]; !exists {
				diff.ForeignKeysToDrop = append(diff.ForeignKeysToDrop, currentRel)
			}
		}
	}

	return diff
}

// normalizeFieldMetadata normalizes field metadata for comparison, ignoring GORM-specific metadata that doesn't affect DB schema
func normalizeFieldMetadata(field *schema.Field) *schema.Field {
	if field == nil {
		return nil
	}

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
	if !strings.EqualFold(a.DBName, b.DBName) {
		return false
	}

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

	if normalizeDefaultValue(a.DefaultValue) != normalizeDefaultValue(b.DefaultValue) {
		return false
	}

	if a.AutoIncrement != b.AutoIncrement {
		return false
	}

	if !a.PrimaryKey && a.NotNull != b.NotNull {
		return false
	}

	return true
}

// normalizeDBType normalizes Go/GORM/Postgres types for DB comparison
func normalizeDBType(dt schema.DataType) string {
	dtStr := strings.ToLower(string(dt))
	if dtStr == "int" || dtStr == "int32" || dtStr == "int4" || dtStr == "int64" || dtStr == "int8" || dtStr == "uint" || dtStr == "bigint" {
		return "bigint"
	}
	if dtStr == "float64" || dtStr == "float32" || dtStr == "float" || dtStr == "real" || dtStr == "numeric" || dtStr == "decimal" || strings.HasPrefix(dtStr, "decimal(") || dtStr == "float8" || dtStr == "double precision" {
		return "decimal"
	}
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

	dv = strings.Trim(dv, "'\"")
	dv = strings.ToLower(dv)

	if strings.HasPrefix(dv, "nextval") {
		return "auto_increment"
	}

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

func relationshipsEqual(source, target *schema.Relationship) bool {
	if source == nil || target == nil {
		return false
	}
	if source.Field == nil || target.Field == nil {
		return false
	}
	if source.Field.Schema == nil || target.Field.Schema == nil {
		return false
	}

	if source.Field.Schema.Table != target.Field.Schema.Table {
		return false
	}

	if len(source.References) != len(target.References) {
		return false
	}
	return true
}
