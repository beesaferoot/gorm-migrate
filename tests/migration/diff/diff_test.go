package migration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gorm-schema/internal/migration/diff"
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
}
