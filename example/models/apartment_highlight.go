package models

import "gorm.io/gorm"

// ApartmentHighlight represents a feature or highlight of an apartment
type ApartmentHighlight struct {
	gorm.Model
	ApartmentID int
	Apartment   *Apartment `gorm:"foreignKey:ApartmentID"`
	Feature     string
}
