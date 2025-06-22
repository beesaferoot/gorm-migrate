package diff

import (
	"fmt"
	"reflect"
	"strings"

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
	// First pass: parse all models and collect their schemas
	for _, model := range models {
		stmt := &gorm.Statement{DB: c.db}
		if err := stmt.Parse(model); err != nil {
			return nil, err
		}
		s := stmt.Schema
		// Only include fields that map to DB columns
		columns := make([]*schema.Field, 0)
		seenColumns := make(map[string]bool) // Track seen column names to handle embedded structs
		for _, field := range s.Fields {
			if field.DBName == "" {
				continue // skip non-column fields
			}
			// Skip if we've already seen this column name (from an embedded struct)
			if seenColumns[field.DBName] {
				continue
			}
			seenColumns[field.DBName] = true
			columns = append(columns, field)
		}
		// Create a shallow copy of schema with only column fields
		copySchema := *s
		copySchema.Fields = columns
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
		s.Relationships = relationships
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

	// Use GORM's Migrator to get the current database schema
	migrator := db.Migrator()

	// Get all tables from the database
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

		// Create a temporary model instance to get the schema
		// We'll use a generic struct that GORM can introspect
		tempModel := &struct {
			ID uint `gorm:"primaryKey"`
		}{}

		// Set the table name for this temporary model
		stmt := &gorm.Statement{DB: db, Table: tableName}
		if err := stmt.Parse(tempModel); err != nil {
			// If parsing fails, skip this table
			continue
		}

		// Get the schema from the statement
		tableSchema := stmt.Schema
		if tableSchema == nil {
			continue
		}

		// Get column information using Migrator
		columns, err := migrator.ColumnTypes(tableName)
		if err != nil {
			continue
		}

		// Build fields from column information
		var fields []*schema.Field
		for _, col := range columns {
			// Get column properties with proper error handling
			isPrimaryKey, _ := col.PrimaryKey()
			isAutoIncrement, _ := col.AutoIncrement()
			defaultValue, _ := col.DefaultValue()
			length, _ := col.Length()
			precision, scale, _ := col.DecimalSize()
			nullable, _ := col.Nullable()

			field := &schema.Field{
				Name:          col.Name(),
				DBName:        col.Name(),
				DataType:      schema.DataType(col.DatabaseTypeName()),
				NotNull:       !nullable, // Nullable() returns true if nullable, so we invert it
				PrimaryKey:    isPrimaryKey,
				AutoIncrement: isAutoIncrement,
				DefaultValue:  defaultValue,
				Size:          int(length),
				Precision:     int(precision),
				Scale:         int(scale),
				// Set proper metadata to match GORM expectations
				Creatable: true,
				Updatable: true,
				Readable:  true,
			}
			fields = append(fields, field)
		}

		// Create the schema
		schemas[tableName] = &schema.Schema{
			Name:   tableName,
			Table:  tableName,
			Fields: fields,
		}
	}

	return schemas, nil
}

func (c *SchemaComparer) getCurrentSchemaByDialect(db *gorm.DB) (map[string]*schema.Schema, error) {
	//nolint:staticcheck // explicit selector for clarity
	switch db.Dialector.Name() {
	case "sqlite":
		return c.getSQLiteSchema(db)
	case "postgres":
		return c.getPostgresSchema(db)
	default:
		return nil, fmt.Errorf("getCurrentSchema only implemented for sqlite and postgres")
	}
}

func (c *SchemaComparer) getPostgresSchema(db *gorm.DB) (map[string]*schema.Schema, error) {
	tables := make(map[string]*schema.Schema)

	// Get all tables
	rows, err := db.Raw(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
	`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}

		// Get column info with correct primary key detection
		colRows, err := db.Raw(`
			SELECT 
				c.column_name, 
				c.data_type, 
				c.is_nullable, 
				c.column_default,
				(CASE WHEN kcu.column_name IS NOT NULL THEN true ELSE false END) as is_primary_key,
				(CASE WHEN c.column_default LIKE 'nextval%' OR c.column_default LIKE 'gen_random_uuid%' THEN true ELSE false END) as is_auto_increment
			FROM information_schema.columns c
			LEFT JOIN information_schema.table_constraints tc 
				ON tc.table_name = c.table_name 
				AND tc.constraint_type = 'PRIMARY KEY'
				AND tc.table_schema = c.table_schema
			LEFT JOIN information_schema.key_column_usage kcu 
				ON kcu.constraint_name = tc.constraint_name 
				AND kcu.column_name = c.column_name
				AND kcu.table_name = c.table_name
				AND kcu.table_schema = c.table_schema
			WHERE c.table_name = ? AND c.table_schema = 'public'
			ORDER BY c.ordinal_position
		`, tableName).Rows()
		if err != nil {
			return nil, err
		}

		fields := make([]*schema.Field, 0)
		for colRows.Next() {
			var (
				name       string
				typeStr    string
				nullable   string
				defaultVal interface{}
				isPK       bool
				isAutoInc  bool
			)
			if err := colRows.Scan(&name, &typeStr, &nullable, &defaultVal, &isPK, &isAutoInc); err != nil {
				colRows.Close()
				return nil, err
			}

			goFieldName := toExportedFieldName(name)
			goType := postgresTypeToGoType(typeStr)
			field := &schema.Field{
				Name:          goFieldName,
				DBName:        name,
				DataType:      schema.DataType(goType),
				NotNull:       nullable == "NO",
				PrimaryKey:    isPK,
				AutoIncrement: isAutoInc,
				DefaultValue:  fmt.Sprintf("%v", defaultVal),
				// Set proper metadata to match GORM expectations
				Creatable: true,
				Updatable: true,
				Readable:  true,
			}
			fields = append(fields, field)
		}
		colRows.Close()

		// Get foreign key info
		fkRows, err := db.Raw(`
			SELECT
				kcu.column_name,
				ccu.table_name AS foreign_table_name,
				ccu.column_name AS foreign_column_name
			FROM information_schema.table_constraints AS tc
			JOIN information_schema.key_column_usage AS kcu
				ON tc.constraint_name = kcu.constraint_name
			JOIN information_schema.constraint_column_usage AS ccu
				ON ccu.constraint_name = tc.constraint_name
			WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_name = ?
		`, tableName).Rows()
		if err != nil {
			return nil, err
		}

		relationships := schema.Relationships{}
		for fkRows.Next() {
			var (
				columnName        string
				foreignTableName  string
				foreignColumnName string
			)
			if err := fkRows.Scan(&columnName, &foreignTableName, &foreignColumnName); err != nil {
				fkRows.Close()
				return nil, err
			}

			// Add belongs-to relationship
			relationships.BelongsTo = append(relationships.BelongsTo, &schema.Relationship{
				Type: schema.BelongsTo,
				Field: &schema.Field{
					Name:   toExportedFieldName(columnName),
					DBName: columnName,
				},
			})
		}
		fkRows.Close()

		// Create schema
		tables[tableName] = &schema.Schema{
			Name:          toExportedFieldName(tableName),
			Table:         tableName,
			Fields:        fields,
			Relationships: relationships,
		}
	}

	return tables, nil
}

func postgresTypeToGoType(pgType string) string {
	switch pgType {
	case "integer", "int", "int4":
		return "int"
	case "bigint", "int8":
		return "int64"
	case "smallint", "int2":
		return "int"
	case "character varying", "varchar", "text", "char", "character":
		return "string"
	case "boolean", "bool":
		return "bool"
	case "numeric", "decimal":
		return "float64"
	case "real", "float4":
		return "float32"
	case "double precision", "float8":
		return "float64"
	case "timestamp with time zone", "timestamp without time zone", "date", "time":
		return "time.Time"
	case "json", "jsonb":
		return "json"
	case "uuid":
		return "string"
	default:
		return "string"
	}
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

	// Compare fields
	currentFields := make(map[string]*schema.Field)
	for _, field := range current.Fields {
		// Skip relationship fields (not actual DB columns)
		if isRelationshipField(field) {
			continue
		}
		// Normalize field metadata for comparison
		normalizedField := normalizeFieldMetadata(field)
		currentFields[field.DBName] = normalizedField
	}

	targetFields := make(map[string]*schema.Field)
	for _, field := range target.Fields {
		// Skip relationship fields (not actual DB columns)
		if isRelationshipField(field) {
			continue
		}
		// Normalize field metadata for comparison
		normalizedField := normalizeFieldMetadata(field)
		targetFields[field.DBName] = normalizedField
	}

	// Fields to add or modify
	for dbName, targetField := range targetFields {
		if currentField, exists := currentFields[dbName]; !exists {
			diff.FieldsToAdd = append(diff.FieldsToAdd, targetField)
		} else if !fieldsEqual(currentField, targetField) {
			if debugDiffOutput {
				fmt.Printf("[DEBUG] Field modification detected for %s.%s: current type=%v, target type=%v\n", target.Table, targetField.DBName, currentField.DataType, targetField.DataType)
			}
			diff.FieldsToModify = append(diff.FieldsToModify, targetField)
		}
	}
	// Ignore fields to drop (orphaned columns in DB)
	// for dbName, currentField := range currentFields {
	// 	if _, exists := targetFields[dbName]; !exists {
	// 		diff.FieldsToDrop = append(diff.FieldsToDrop, currentField)
	// 	}
	// }

	// Ignore foreign key diffs for pure model-driven schema
	// (do not populate diff.ForeignKeysToAdd or diff.ForeignKeysToDrop)

	// Indexes (by name)
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

// relationshipsEqual compares two *schema.Relationship for relevant diff purposes
func relationshipsEqual(a, b *schema.Relationship) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Field == nil || b.Field == nil {
		return a.Field == b.Field
	}
	return a.Field.DBName == b.Field.DBName && a.Schema != nil && b.Schema != nil && a.Schema.Table == b.Schema.Table
}

func tableNames(schemas map[string]*schema.Schema) []string {
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	return names
}

func fieldNames(fields []*schema.Field) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}

// Helper to convert snake_case or lower to ExportedCamelCase
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

// Map SQLite type to Go type string
func sqliteTypeToGoType(sqliteType string) string {
	t := sqliteType
	t = lower(t)
	switch {
	case contains(t, "int"):
		return "int"
	case contains(t, "char"), contains(t, "clob"), contains(t, "text"):
		return "string"
	case contains(t, "blob"):
		return "[]byte"
	case contains(t, "real"), contains(t, "floa"), contains(t, "doub"):
		return "float64"
	case contains(t, "bool"):
		return "bool"
	case contains(t, "date"), contains(t, "time"):
		return "time.Time"
	}
	return "string"
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] = b[i] - 'A' + 'a'
		}
	}
	return string(b)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func (c *SchemaComparer) getSQLiteSchema(db *gorm.DB) (map[string]*schema.Schema, error) {
	tables := make(map[string]*schema.Schema)
	rows, err := db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}

		// Get column info
		colRows, err := db.Raw(fmt.Sprintf("PRAGMA table_info('%s')", tableName)).Rows()
		if err != nil {
			return nil, err
		}

		fields := make([]*schema.Field, 0)
		for colRows.Next() {
			var (
				cid        int
				name       string
				typeStr    string
				notnull    int
				defaultVal interface{}
				pk         int
			)
			if err := colRows.Scan(&cid, &name, &typeStr, &notnull, &defaultVal, &pk); err != nil {
				colRows.Close()
				return nil, err
			}

			goFieldName := toExportedFieldName(name)
			goType := sqliteTypeToGoType(typeStr)
			field := &schema.Field{
				Name:         goFieldName,
				DBName:       name,
				DataType:     schema.DataType(goType),
				NotNull:      notnull == 1,
				PrimaryKey:   pk == 1,
				DefaultValue: fmt.Sprintf("%v", defaultVal),
			}
			fields = append(fields, field)
		}
		colRows.Close()

		// Get index info
		indexRows, err := db.Raw(fmt.Sprintf("PRAGMA index_list('%s')", tableName)).Rows()
		if err != nil {
			return nil, err
		}
		indexes := make(map[string]bool)
		for indexRows.Next() {
			var (
				seq     int
				name    string
				unique  int
				origin  string
				partial int
			)
			if err := indexRows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
				indexRows.Close()
				return nil, err
			}
			indexes[name] = unique == 1
		}
		indexRows.Close()

		// Get index columns
		for indexName, isUnique := range indexes {
			indexColRows, err := db.Raw(fmt.Sprintf("PRAGMA index_info('%s')", indexName)).Rows()
			if err != nil {
				return nil, err
			}
			for indexColRows.Next() {
				var (
					seqno   int
					cid     int
					colName string
				)
				if err := indexColRows.Scan(&seqno, &cid, &colName); err != nil {
					indexColRows.Close()
					return nil, err
				}
				// Find the field and mark it as unique if the index is unique
				for _, field := range fields {
					if field.DBName == colName {
						field.Unique = isUnique
						break
					}
				}
			}
			indexColRows.Close()
		}

		// Get foreign key info
		fkRows, err := db.Raw(fmt.Sprintf("PRAGMA foreign_key_list('%s')", tableName)).Rows()
		if err != nil {
			return nil, err
		}
		relationships := schema.Relationships{}
		for fkRows.Next() {
			var (
				id       int
				seq      int
				table    string
				from     string
				to       string
				onUpdate string
				onDelete string
				match    string
			)
			if err := fkRows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				fkRows.Close()
				return nil, err
			}
			// Find the field and add a BelongsTo relationship
			for _, field := range fields {
				if field.DBName == from {
					relationships.BelongsTo = append(relationships.BelongsTo, &schema.Relationship{
						Field: field,
						Type:  schema.BelongsTo,
					})
					break
				}
			}
		}
		fkRows.Close()

		// Build schema.Schema
		parsedSchema := &schema.Schema{
			Name:          toExportedFieldName(tableName),
			Table:         tableName,
			Fields:        fields,
			Relationships: relationships,
		}
		tables[tableName] = parsedSchema
	}

	return tables, nil
}

// Helper to convert CamelCase to snake_case
func toSnakeCase(str string) string {
	if str == "" {
		return ""
	}
	var result []rune
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, toLowerRune(r))
	}
	return string(result)
}

func toLowerRune(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r - 'A' + 'a'
	}
	return r
}
