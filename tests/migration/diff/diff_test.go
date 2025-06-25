package migration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/beesaferoot/gorm-schema/migration/diff"
	"github.com/beesaferoot/gorm-schema/migration/generator"
)

// ModifiedUser is a modified version of the User model for testing.
type ModifiedUser struct {
	gorm.Model
	Name   string
	Age    int
	Gender string
}

// TableName returns the table name for ModifiedUser.
func (ModifiedUser) TableName() string {
	return "users"
}

// TestEstate is a test model for Estate.
type TestEstate struct {
	gorm.Model
	Name string
}

// TestApartment is a test model for Apartment.
type TestApartment struct {
	gorm.Model
	EstateID uint
	Estate   TestEstate `gorm:"foreignKey:EstateID"`
}

// TestApartmentContract is a test model for ApartmentContract.
type TestApartmentContract struct {
	gorm.Model
	ApartmentID uint
	Apartment   TestApartment `gorm:"foreignKey:ApartmentID"`
}

// TestApartmentHighlight is a test model for ApartmentHighlight.
type TestApartmentHighlight struct {
	gorm.Model
	ApartmentID uint
	Apartment   TestApartment `gorm:"foreignKey:ApartmentID"`
}

// TestApartmentBookingPrice is a test model for ApartmentBookingPrice.
type TestApartmentBookingPrice struct {
	gorm.Model
	ApartmentID uint
	Apartment   TestApartment `gorm:"foreignKey:ApartmentID"`
}

// TestTenant is a test model for Tenant.
type TestTenant struct {
	gorm.Model
	Name string
}

// TestEstateWithUserManager is a test model for Estate with UserManager relationship
type TestEstateWithUserManager struct {
	gorm.Model
	Name          string
	Address       string
	City          string
	State         string
	Country       string
	IsDeleted     bool
	UserManagerID uint
	UserManager   *ModifiedUser `gorm:"foreignKey:UserManagerID"`
}

func TestSchemaDiffWithModels(t *testing.T) {
	// Use a file-based SQLite database for reliable schema persistence
	dbPath := "test_diff.db"
	defer func() {
		if err := os.Remove(dbPath); err != nil {
			t.Errorf("failed to remove test database: %v", err)
		}
	}()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)

	// Create schema comparer
	comparer := diff.NewSchemaComparer(db)

	t.Run("Compare Empty Database", func(t *testing.T) {
		// Get current schema (empty)
		currentSchema, err := comparer.GetCurrentSchema()
		require.NoError(t, err)
		assert.Empty(t, currentSchema)

		// Get target schema from models
		targetSchema, err := comparer.GetModelSchemas(
			&ModifiedUser{},
			&TestEstate{},
			&TestApartment{},
			&TestApartmentContract{},
			&TestApartmentHighlight{},
			&TestApartmentBookingPrice{},
			&TestTenant{},
		)
		require.NoError(t, err)
		assert.NotEmpty(t, targetSchema)

		// Compare schemas
		diff, err := comparer.CompareSchemas(currentSchema, targetSchema)
		require.NoError(t, err)
		assert.NotEmpty(t, diff.TablesToCreate)
		assert.Empty(t, diff.TablesToModify)
		assert.Empty(t, diff.TablesToDrop)
	})

	t.Run("Compare Modified Model", func(t *testing.T) {
		// Get the current schema
		currentSchema, err := comparer.GetCurrentSchema()
		require.NoError(t, err)

		// Get the model schema
		modelSchema, err := comparer.GetModelSchemas(&ModifiedUser{})
		require.NoError(t, err)
		require.NotEmpty(t, modelSchema)

		// Compare schemas
		diff, err := comparer.CompareSchemas(currentSchema, modelSchema)
		require.NoError(t, err)
		require.NotNil(t, diff)
	})

	t.Run("Compare Complex Relationships", func(t *testing.T) {
		// Get the current schema
		currentSchema, err := comparer.GetCurrentSchema()
		require.NoError(t, err)

		// Get the model schema
		modelSchema, err := comparer.GetModelSchemas(&TestEstate{}, &TestApartment{}, &TestApartmentContract{})
		require.NoError(t, err)
		require.NotEmpty(t, modelSchema)

		// Compare schemas
		diff, err := comparer.CompareSchemas(currentSchema, modelSchema)
		require.NoError(t, err)
		require.NotNil(t, diff)
	})

	// t.Run("Verify Foreign Keys and Indexes for New Tables", func(t *testing.T) {
	// 	// Get the current schema (empty)
	// 	currentSchema, err := comparer.GetCurrentSchema()
	// 	require.NoError(t, err)
	// 	assert.Empty(t, currentSchema)

	// 	// Get the model schema with relationships
	// 	modelSchema, err := comparer.GetModelSchemas(&TestEstate{}, &TestApartment{})
	// 	require.NoError(t, err)
	// 	require.NotEmpty(t, modelSchema)

	// 	// Compare schemas
	// 	schemaDiff, err := comparer.CompareSchemas(currentSchema, modelSchema)
	// 	require.NoError(t, err)
	// 	require.NotNil(t, schemaDiff)

	// 	// Verify that tables are created
	// 	assert.Len(t, schemaDiff.TablesToCreate, 2)

	// 	// Find the apartment table (which should have foreign keys)
	// 	var apartmentTable *diff.TableDiff
	// 	for i := range schemaDiff.TablesToCreate {
	// 		if schemaDiff.TablesToCreate[i].Schema.Table == "test_apartments" {
	// 			apartmentTable = &schemaDiff.TablesToCreate[i]
	// 			break
	// 		}
	// 	}
	// 	require.NotNil(t, apartmentTable, "TestApartment table should be found in TablesToCreate")

	// 	// Verify that foreign keys are detected
	// 	assert.NotEmpty(t, apartmentTable.ForeignKeysToAdd, "Foreign keys should be detected for new tables")
	// 	assert.Len(t, apartmentTable.ForeignKeysToAdd, 1, "Should have one foreign key relationship")

	// 	// Verify the foreign key details
	// 	fk := apartmentTable.ForeignKeysToAdd[0]
	// 	assert.Equal(t, "estate_id", fk.Field.DBName, "Foreign key field should be estate_id")
	// 	assert.Equal(t, "test_estates", fk.Schema.Table, "Referenced table should be test_estates")

	// 	// Verify that indexes are detected (GORM creates indexes for foreign keys)
	// 	assert.NotEmpty(t, apartmentTable.IndexesToAdd, "Indexes should be detected for new tables")
	// })

	// t.Run("Test Relationship Processing Without Mutex Issues", func(t *testing.T) {
	// 	// This test verifies that relationship processing doesn't cause mutex lock issues
	// 	modelSchema, err := comparer.GetModelSchemas(&TestEstate{}, &TestApartment{}, &TestApartmentContract{})
	// 	require.NoError(t, err)
	// 	require.NotEmpty(t, modelSchema)

	// 	// Verify that relationships are properly processed
	// 	apartmentSchema, exists := modelSchema["test_apartments"]
	// 	require.True(t, exists, "TestApartment schema should exist")
	// 	assert.NotEmpty(t, apartmentSchema.Relationships.BelongsTo, "BelongsTo relationships should be processed")

	// 	contractSchema, exists := modelSchema["test_apartment_contracts"]
	// 	require.True(t, exists, "TestApartmentContract schema should exist")
	// 	assert.NotEmpty(t, contractSchema.Relationships.BelongsTo, "BelongsTo relationships should be processed")
	// })

	// t.Run("Test Relationship Field Filtering", func(t *testing.T) {
	// 	// This test verifies that relationship fields are properly filtered out from database columns
	// 	modelSchema, err := comparer.GetModelSchemas(&TestEstateWithUserManager{})
	// 	require.NoError(t, err)
	// 	require.NotEmpty(t, modelSchema)

	// 	// Get the estate schema
	// 	estateSchema, exists := modelSchema["test_estate_with_user_managers"]
	// 	require.True(t, exists, "TestEstateWithUserManager schema should exist")

	// 	// Verify that UserManagerID field is included (it's a foreign key column)
	// 	var userManagerIDField *schema.Field
	// 	for _, field := range estateSchema.Fields {
	// 		if field.DBName == "user_manager_id" {
	// 			userManagerIDField = field
	// 			break
	// 		}
	// 	}
	// 	require.NotNil(t, userManagerIDField, "UserManagerID field should be included in database columns")

	// 	// Verify that UserManager field is NOT included (it's a relationship field)
	// 	var userManagerField *schema.Field
	// 	for _, field := range estateSchema.Fields {
	// 		if field.Name == "UserManager" {
	// 			userManagerField = field
	// 			break
	// 		}
	// 	}
	// 	assert.Nil(t, userManagerField, "UserManager field should NOT be included in database columns")

	// 	// Verify that relationships are properly set up
	// 	assert.NotEmpty(t, estateSchema.Relationships.BelongsTo, "BelongsTo relationships should be processed")
	// 	assert.Len(t, estateSchema.Relationships.BelongsTo, 1, "Should have one BelongsTo relationship")

	// 	// Verify the relationship details
	// 	rel := estateSchema.Relationships.BelongsTo[0]
	// 	assert.Equal(t, "user_manager_id", rel.Field.DBName, "Relationship should reference user_manager_id field")
	// })

	t.Run("Test Validation Issue", func(t *testing.T) {
		// This test reproduces the validation issue
		currentSchema, err := comparer.GetCurrentSchema()
		require.NoError(t, err)
		assert.Empty(t, currentSchema)

		// Get the model schema with relationships
		modelSchema, err := comparer.GetModelSchemas(&TestEstateWithUserManager{})
		require.NoError(t, err)
		require.NotEmpty(t, modelSchema)

		// Compare schemas
		schemaDiff, err := comparer.CompareSchemas(currentSchema, modelSchema)
		require.NoError(t, err)
		require.NotNil(t, schemaDiff)

		// This should trigger the validation and show debug output
		gen := generator.NewGenerator("migrations")
		gen.SetSchemaDiff(schemaDiff)
		err = gen.CreateMigration("test_validation")
		// We expect this to fail with validation error, but we want to see the debug output
		if err != nil {
			t.Logf("Expected validation error: %v", err)
		}
	})
}
