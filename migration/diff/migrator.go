package diff

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Migrator interface {
	ColumnTypes(dst interface{}) ([]gorm.ColumnType, error)
	GetTables() ([]string, error)
	GetIndexes(tableName string) ([]*schema.Index, error)
	GetRelationships(tableName string) ([]*schema.Relationship, error)
}

type SchemaMigrator struct {
	gormMigrator gorm.Migrator
	db           *gorm.DB
}

func NewSchemaMigrator(db *gorm.DB) Migrator {
	return &SchemaMigrator{
		gormMigrator: db.Migrator(),
		db:           db,
	}
}

func (m *SchemaMigrator) ColumnTypes(dst interface{}) ([]gorm.ColumnType, error) {
	return m.gormMigrator.ColumnTypes(dst)
}

func (m *SchemaMigrator) GetTables() ([]string, error) {
	return m.gormMigrator.GetTables()
}

func (m *SchemaMigrator) GetIndexes(tableName string) ([]*schema.Index, error) {
	// Handle empty table name
	if tableName == "" {
		return []*schema.Index{}, nil
	}

	var indexes []*schema.Index

	// Query to get index information from PostgreSQL system catalogs
	query := `
	SELECT 
		i.indexname,
		i.indexdef,
		ix.indisunique,
		ix.indisprimary,
		array_to_string(array_agg(a.attname ORDER BY t.ordinality), ',') as column_names
	FROM pg_indexes i
	JOIN pg_class c ON c.relname = i.tablename
	JOIN pg_index ix ON ix.indexrelid = (i.schemaname||'.'||i.indexname)::regclass
	JOIN unnest(ix.indkey) WITH ORDINALITY t(attnum, ordinality) ON true
	JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = t.attnum
		WHERE i.tablename = $1
		GROUP BY i.indexname, i.indexdef, ix.indisunique, ix.indisprimary;
	`

	rows, err := m.db.Raw(query, tableName).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var indexName, columnNames string
		var isUnique, isPrimaryKey bool

		if err := rows.Scan(&indexName, &isUnique, &isPrimaryKey, &columnNames); err != nil {
			return nil, fmt.Errorf("failed to scan index row: %w", err)
		}

		// Parse column names
		columns := strings.Split(columnNames, ",")
		var fields []schema.IndexOption
		for _, col := range columns {
			col = strings.TrimSpace(col)
			if col != "" {
				fields = append(fields, schema.IndexOption{
					Field: &schema.Field{DBName: col},
				})
			}
		}

		// Create index
		index := &schema.Index{
			Name:   indexName,
			Type:   "BTREE", // PostgreSQL default index type
			Fields: fields,
			Option: func() string {
				if isUnique {
					return "UNIQUE"
				}
				return ""
			}(),
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}


func (m *SchemaMigrator) GetRelationships(tableName string) ([]*schema.Relationship, error) {
	// Handle empty table name
	if tableName == "" {
		return []*schema.Relationship{}, nil
	}

	var relationships []*schema.Relationship

	// Query to get foreign key information from PostgreSQL information_schema
	query := `
	SELECT 
		tc.constraint_name,
		tc.table_name,
		kcu.column_name,
		ccu.table_name AS referenced_table_name,
		ccu.column_name AS referenced_column_name,
		rc.delete_rule AS on_delete,
		rc.update_rule AS on_update
	FROM 
		information_schema.table_constraints AS tc 
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints AS rc
			ON tc.constraint_name = rc.constraint_name
			AND tc.table_schema = rc.constraint_schema
	WHERE 
		tc.constraint_type = 'FOREIGN KEY' 
		AND tc.table_name = $1
	ORDER BY 
		tc.constraint_name, kcu.ordinal_position;
	`

	rows, err := m.db.Raw(query, tableName).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships for table %s: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var constraintName, tableName, columnName, referencedTableName, referencedColumnName string
		var onDelete, onUpdate string

		if err := rows.Scan(&constraintName, &tableName, &columnName, &referencedTableName, &referencedColumnName, &onDelete, &onUpdate); err != nil {
			return nil, fmt.Errorf("failed to scan foreign key row: %w", err)
		}

		// Create relationship
		relationship := &schema.Relationship{
			Name: constraintName,
			Type: schema.BelongsTo,
			Field: &schema.Field{
				DBName: columnName,
				Schema: &schema.Schema{
					Table: tableName,
				},
			},
			Schema: &schema.Schema{
				Table: referencedTableName,
			},
			References: []*schema.Reference{
				{
					ForeignKey: &schema.Field{
						DBName: columnName,
					},
					PrimaryKey: &schema.Field{
						DBName: referencedColumnName,
					},
				},
			},
		}

		relationships = append(relationships, relationship)
	}

	return relationships, nil
}
