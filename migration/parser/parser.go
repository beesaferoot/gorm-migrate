package parser

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/beesaferoot/gorm-schema/migration"
)

type ModelParser struct {
	db     *gorm.DB
	models map[string]interface{}
}

func NewModelParser(db *gorm.DB) (*ModelParser, error) {
	// Validate that user has provided a registry
	if err := migration.ValidateRegistry(); err != nil {
		return nil, err
	}

	p := &ModelParser{
		db:     db,
		models: migration.GlobalModelRegistry.GetModels(),
	}

	if len(p.models) == 0 {
		return nil, fmt.Errorf("no models found in registry")
	}

	return p, nil
}

func (p *ModelParser) Parse() (map[string]*schema.Schema, error) {
	schemas := make(map[string]*schema.Schema)

	for name, model := range p.models {
		stmt := &gorm.Statement{DB: p.db, Table: strings.ToLower(name)}
		if err := stmt.Parse(model); err != nil {
			return nil, fmt.Errorf("failed to parse model %s with GORM: %w. Check for unsupported field types or incorrect struct tags", name, err)
		}
		mSchema := stmt.Schema
		if mSchema == nil {
			return nil, fmt.Errorf("GORM failed to produce a schema for model %s. This can happen if the model is empty or invalid", name)
		}

		// Ensure schema has proper name and table
		mSchema.Name = name
		mSchema.Table = strings.ToLower(name)

		schemas[name] = mSchema
	}
	return schemas, nil
}
