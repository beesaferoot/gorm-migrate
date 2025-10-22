package models

import (
	"time"

	"gorm.io/gorm"
)

type CustomSchema struct {
	gorm.Model
	Title     string         `json:"title" gorm:"not null" validate:"nonzero"`
	File      string         `json:"file" gorm:"not null"`
	Duration  int            `json:"duration" gorm:"default:0"`
	Thumbnail string         `json:"thumbnail" gorm:"type:text"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (CustomSchema) TableName() string {
	return "schema.custom_table"
}
