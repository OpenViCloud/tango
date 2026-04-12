package models

import "time"

type CloudflareConnectionRecord struct {
	ID                string    `gorm:"primaryKey;type:text"`
	UserID            string    `gorm:"column:user_id;type:text;not null;index"`
	DisplayName       string    `gorm:"column:display_name;type:text;not null"`
	AccountID         string    `gorm:"column:account_id;type:text;not null"`
	ZoneID            string    `gorm:"column:zone_id;type:text;not null"`
	APITokenEncrypted string    `gorm:"column:api_token_encrypted;type:text;not null"`
	Status            string    `gorm:"column:status;type:varchar(32);not null"`
	CreatedAt         time.Time `gorm:"column:created_at;not null"`
	UpdatedAt         time.Time `gorm:"column:updated_at;not null"`
}

func (CloudflareConnectionRecord) TableName() string { return "cloudflare_connections" }
