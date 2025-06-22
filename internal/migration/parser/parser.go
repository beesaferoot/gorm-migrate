package parser

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"gorm-schema/internal/migration"
)

type ModelParser struct {
	db         *gorm.DB
	modelTypes map[string]reflect.Type
}

func NewModelParser(db *gorm.DB) (*ModelParser, error) {
	// Validate that user has provided a registry
	if err := migration.ValidateRegistry(); err != nil {
		return nil, err
	}

	p := &ModelParser{
		db:         db,
		modelTypes: migration.GlobalModelRegistry.GetModelTypes(),
	}

	if len(p.modelTypes) == 0 {
		return nil, fmt.Errorf("no models found in registry")
	}

	return p, nil
}

func (p *ModelParser) Parse() (map[string]*schema.Schema, error) {
	schemas := make(map[string]*schema.Schema)

	for name, modelType := range p.modelTypes {
		instance := reflect.New(modelType).Interface()
		stmt := &gorm.Statement{DB: p.db, Table: strings.ToLower(name)}
		if err := stmt.Parse(instance); err != nil {
			return nil, fmt.Errorf("failed to parse model %s with GORM: %w. Check for unsupported field types or incorrect struct tags", name, err)
		}
		mSchema := stmt.Schema
		if mSchema == nil {
			return nil, fmt.Errorf("GORM failed to produce a schema for model %s. This can happen if the model is empty or invalid", name)
		}

		// Ensure schema has proper name and table
		mSchema.Name = name
		mSchema.Table = strings.ToLower(name)

		// Validate and fix fields
		if err := p.validateAndFixSchema(mSchema); err != nil {
			return nil, fmt.Errorf("failed to validate schema for model %s: %w", name, err)
		}

		schemas[name] = mSchema
	}
	return schemas, nil
}

// validateAndFixSchema ensures the schema is properly formed
func (p *ModelParser) validateAndFixSchema(s *schema.Schema) error {
	if s == nil {
		return fmt.Errorf("schema is nil")
	}

	if s.Table == "" {
		return fmt.Errorf("table name is empty")
	}

	// First, set up foreign key relationships properly
	p.setupForeignKeyRelationships(s)

	// Filter out fields that should not be database columns
	var validFields []*schema.Field
	for _, field := range s.Fields {
		if field == nil {
			continue
		}

		// Skip struct reference fields (these are for GORM relationships, not DB columns)
		if p.isStructReferenceField(field) {
			continue
		}

		// If DBName is empty, use the field name
		if field.DBName == "" {
			field.DBName = strings.ToLower(field.Name)
		}

		// Ensure DataType is set
		if field.DataType == "" {
			field.DataType = p.inferDataType(field)
		}

		validFields = append(validFields, field)
	}

	// Update the schema with only valid fields
	s.Fields = validFields

	return nil
}

// setupForeignKeyRelationships sets up proper foreign key relationships
func (p *ModelParser) setupForeignKeyRelationships(s *schema.Schema) {
	// Create a map of field names to fields for quick lookup
	fieldMap := make(map[string]*schema.Field)
	for _, field := range s.Fields {
		fieldMap[field.Name] = field
	}

	// Process BelongsTo relationships
	var validBelongsTo []*schema.Relationship
	for _, rel := range s.Relationships.BelongsTo {
		if rel.Field != nil && rel.Field.TagSettings != nil {
			// Get the foreign key field name from the GORM tag
			if fkFieldName, exists := rel.Field.TagSettings["FOREIGNKEY"]; exists {
				// Find the actual foreign key field
				if fkField, exists := fieldMap[fkFieldName]; exists {
					// Find the referenced schema by looking up the model type
					var referencedSchema *schema.Schema
					if rel.FieldSchema != nil {
						// Look up the referenced schema by name
						for name, modelType := range p.modelTypes {
							if name == rel.FieldSchema.Name {
								// Create a temporary instance to get the schema
								instance := reflect.New(modelType).Interface()
								stmt := &gorm.Statement{DB: p.db, Table: strings.ToLower(name)}
								if err := stmt.Parse(instance); err == nil && stmt.Schema != nil {
									referencedSchema = stmt.Schema
									break
								}
							}
						}
					}

					// Create a new relationship with the correct foreign key field and referenced schema
					newRel := &schema.Relationship{
						Type:        schema.BelongsTo,
						Field:       fkField,
						Schema:      referencedSchema,
						FieldSchema: rel.FieldSchema,
					}
					validBelongsTo = append(validBelongsTo, newRel)
				}
			}
		}
	}
	s.Relationships.BelongsTo = validBelongsTo
}

// isStructReferenceField checks if a field is a struct reference that should not be a database column
func (p *ModelParser) isStructReferenceField(field *schema.Field) bool {
	if field == nil {
		return false
	}

	// Check if it's a struct type (not a basic type)
	if field.FieldType.Kind() == reflect.Struct {
		// Skip gorm.Model embedded struct
		if field.FieldType.String() == "gorm.Model" {
			return false
		}

		// Check if it's a pointer to a struct (relationship field)
		if field.FieldType.Kind() == reflect.Ptr && field.FieldType.Elem().Kind() == reflect.Struct {
			return true
		}

		// Check if it's a direct struct (not embedded)
		if field.FieldType.Kind() == reflect.Struct {
			// Skip embedded structs that should be database columns
			return field.FieldType.String() == "time.Time"
		}
	}

	// Check if the field name suggests it's a relationship (not an ID field)
	fieldName := strings.ToLower(field.Name)
	if !strings.HasSuffix(fieldName, "_id") && !strings.HasSuffix(fieldName, "id") {
		// Check if it's a common relationship field name
		relationshipNames := []string{"apartment", "tenant", "user", "estate", "bookingprice", "contract", "highlight"}
		for _, relName := range relationshipNames {
			if fieldName == relName {
				return true
			}
		}
	}

	return false
}

// inferDataType infers the GORM data type from the field
func (p *ModelParser) inferDataType(field *schema.Field) schema.DataType {
	if field == nil {
		return schema.DataType("string")
	}

	// Handle time.Time specially
	if field.FieldType.String() == "time.Time" {
		return schema.DataType("time")
	}

	// Handle pointer to time.Time
	if field.FieldType.String() == "*time.Time" {
		return schema.DataType("time")
	}

	// Handle slices
	if field.FieldType.Kind() == reflect.Slice {
		elemType := field.FieldType.Elem()
		switch elemType.Kind() {
		case reflect.String:
			return schema.DataType("json")
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return schema.DataType("json")
		case reflect.Float32, reflect.Float64:
			return schema.DataType("json")
		default:
			return schema.DataType("json")
		}
	}

	// Handle basic types
	switch field.FieldType.Kind() {
	case reflect.String:
		return schema.DataType("varchar")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return schema.DataType("int")
	case reflect.Int64:
		return schema.DataType("bigint")
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return schema.DataType("uint")
	case reflect.Uint64:
		return schema.DataType("bigint")
	case reflect.Float32:
		return schema.DataType("float")
	case reflect.Float64:
		return schema.DataType("double")
	case reflect.Bool:
		return schema.DataType("bool")
	case reflect.Struct:
		return schema.DataType("json")
	default:
		return schema.DataType("string")
	}
}
