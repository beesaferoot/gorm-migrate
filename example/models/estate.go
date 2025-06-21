package models

import "gorm.io/gorm"

// Estate represents a real estate property
type Estate struct {
	gorm.Model
	Name          string
	Address       string
	City          string
	State         string
	Country       string
	IsDeleted     bool
	UserManagerID uint
	UserManager   *User `gorm:"foreignKey:UserManagerID"`
}
