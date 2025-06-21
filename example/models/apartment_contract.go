package models

import (
	"time"

	"gorm.io/gorm"
)

// ApartmentContract represents a rental contract for an apartment
type ApartmentContract struct {
	gorm.Model
	ApartmentID             int
	Apartment               *Apartment `gorm:"foreignKey:ApartmentID"`
	ApartmentBookingPriceID int
	ApartmentBookingPrice   *ApartmentBookingPrice `gorm:"foreignKey:ApartmentBookingPriceID"`
	TenantID                int
	Tenant                  *Tenant `gorm:"foreignKey:TenantID"`
	ContractStart           time.Time
	ContractEnd             time.Time
	CreatedBy               string
	IsDeleted               bool
	DeletedAt               *time.Time
	BookingType             uint
	TotalCost               float64 `gorm:"type:decimal(10,2)"`
	PaymentMode             string
	Email                   string
}
