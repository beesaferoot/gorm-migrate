package models

import (
	"time"

	"gorm.io/gorm"
)

// ApartmentBookingPrice represents the pricing for an apartment
type ApartmentBookingPrice struct {
	gorm.Model
	ApartmentID       int
	Apartment         *Apartment `gorm:"foreignKey:ApartmentID"`
	BookingPrice      float64    `gorm:"type:decimal(10,2)"`
	BookingType       uint
	BookingMultiplier int
	CreatedBy         string
	IsDeleted         bool
	DeletedAt         *time.Time
}
