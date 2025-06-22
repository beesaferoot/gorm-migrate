package models

import "gorm.io/gorm"

// User represents a user in the system
type User struct {
	gorm.Model
	Name  string `gorm:"not null"`
	Email string `gorm:"uniqueIndex;not null"`
	Age   int
}

// Post represents a blog post
type Post struct {
	gorm.Model
	Title   string `gorm:"not null"`
	Content string
	UserID  uint
	User    User `gorm:"foreignKey:UserID"`
}
