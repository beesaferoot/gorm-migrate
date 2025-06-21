package models

import (
	"time"

	"gorm.io/gorm"
)

// Apartment represents an apartment unit within an estate
type Apartment struct {
	gorm.Model
	EstateID               int
	Estate                 *Estate `gorm:"foreignKey:EstateID"`
	ApartmentNumber        string
	Street                 string
	ElectricityMeterNumber string
	IsDeleted              bool
	DeletedAt              *time.Time
	ImageUrls              []string `gorm:"type:json"`
	IsVacant               bool
	Address                string
	BedroomCount           int
	BathroomCount          int
	GuestCount             int
	Size                   float64
	Description            string
}
