package models

import "time"

type BaseDomainRecord struct {
	ID              string    `gorm:"primaryKey;type:text"`
	Domain          string    `gorm:"column:domain;type:text;not null;uniqueIndex"`
	WildcardEnabled bool      `gorm:"column:wildcard_enabled;not null;default:false"`
	CreatedAt       time.Time `gorm:"column:created_at;not null"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null"`
}

func (BaseDomainRecord) TableName() string { return "base_domains" }
