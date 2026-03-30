package models

import "time"

type PlatformConfigRecord struct {
	Key       string    `gorm:"primaryKey;type:text"`
	Value     string    `gorm:"column:value;type:text;not null;default:''"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

func (PlatformConfigRecord) TableName() string { return "platform_configs" }
