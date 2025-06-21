package models

import (
	"time"

	"gorm.io/gorm"
)

// Tenant represents a tenant renting an apartment
type Tenant struct {
	gorm.Model
	FirstName     string
	LastName      string
	Email         string
	Phone         string
	ApartmentID   int
	Apartment     *Apartment `gorm:"foreignKey:ApartmentID"`
	CreatedBy     string
	IdentityDoc   string
	Signature     string
	IsDeleted     bool
	DeletedAt     *time.Time
	OwnerID       int
	TenantType    uint
	ContractStart time.Time
	ContractEnd   time.Time
}
