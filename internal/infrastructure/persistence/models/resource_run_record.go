package models

import "time"

type ResourceRunRecord struct {
	ID         string     `gorm:"primaryKey;type:varchar(64)"`
	ResourceID string     `gorm:"column:resource_id;type:text;not null;index"`
	Status     string     `gorm:"column:status;type:varchar(32);not null;index"`
	Logs       string     `gorm:"column:logs;type:text"`
	ErrorMsg   string     `gorm:"column:error_msg;type:text"`
	StartedAt  *time.Time `gorm:"column:started_at"`
	FinishedAt *time.Time `gorm:"column:finished_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;not null"`
}

func (ResourceRunRecord) TableName() string {
	return "resource_runs"
}
