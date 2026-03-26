package models

import "time"

type ProjectRecord struct {
	ID          string    `gorm:"primaryKey;type:text"`
	Name        string    `gorm:"not null"`
	Description string
	CreatedBy   string    `gorm:"column:created_by;type:text"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null"`
}

func (ProjectRecord) TableName() string { return "projects" }
