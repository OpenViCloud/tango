package models

import "time"

type SourceConnectionRecord struct {
	ID                string     `gorm:"primaryKey;type:text"`
	UserID            string     `gorm:"column:user_id;type:text;not null;index"`
	SourceProviderID  string     `gorm:"column:source_provider_id;type:text;not null;index"`
	Provider          string     `gorm:"column:provider;type:varchar(32);not null;index"`
	DisplayName       string     `gorm:"column:display_name;not null"`
	AccountIdentifier string     `gorm:"column:account_identifier;not null;index"`
	ExternalID        string     `gorm:"column:external_id;not null;index"`
	MetadataJSON      string     `gorm:"column:metadata_json;type:text;not null;default:'{}'"`
	Status            string     `gorm:"column:status;type:varchar(32);not null"`
	ExpiresAt         *time.Time `gorm:"column:expires_at"`
	LastUsedAt        *time.Time `gorm:"column:last_used_at"`
	CreatedAt         time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;not null"`
}

func (SourceConnectionRecord) TableName() string { return "source_connections" }
