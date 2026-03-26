package models

import "time"

type ChannelRecord struct {
	ID                   string     `gorm:"primaryKey;type:text"`
	Name                 string     `gorm:"column:name;not null;uniqueIndex:idx_channels_name"`
	Kind                 string     `gorm:"column:kind;not null;index:idx_channels_kind_status"`
	Status               string     `gorm:"column:status;not null;index:idx_channels_kind_status"`
	EncryptedCredentials string     `gorm:"column:encrypted_credentials"`
	SettingsJSON         string     `gorm:"column:settings_json;not null"`
	CreatedAt            time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;not null"`
	DeletedAt            *time.Time `gorm:"column:deleted_at;index"`
}

func (ChannelRecord) TableName() string {
	return "channels"
}
