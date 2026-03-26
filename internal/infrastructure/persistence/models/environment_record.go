package models

import "time"

type EnvironmentRecord struct {
	ID        string    `gorm:"primaryKey;type:text"`
	Name      string    `gorm:"not null"`
	ProjectID string    `gorm:"column:project_id;type:text;not null;index"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (EnvironmentRecord) TableName() string { return "environments" }
