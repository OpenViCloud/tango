package models

import "time"

type SourceProviderRecord struct {
	ID                   string    `gorm:"primaryKey;type:text"`
	UserID               string    `gorm:"column:user_id;type:text;not null;index"`
	Provider             string    `gorm:"column:provider;type:varchar(32);not null;index"`
	DisplayName          string    `gorm:"column:display_name;not null"`
	EncryptedCredentials string    `gorm:"column:encrypted_credentials;type:text;not null"`
	MetadataJSON         string    `gorm:"column:metadata_json;type:text;not null;default:'{}'"`
	Status               string    `gorm:"column:status;type:varchar(32);not null"`
	CreatedAt            time.Time `gorm:"column:created_at;not null"`
	UpdatedAt            time.Time `gorm:"column:updated_at;not null"`
}

func (SourceProviderRecord) TableName() string { return "source_providers" }
