package models

import "time"

type APIKeyRecord struct {
	ID         string     `gorm:"primaryKey;type:text"`
	Name       string     `gorm:"column:name;not null"`
	KeyHash    string     `gorm:"column:key_hash;not null;uniqueIndex:idx_api_keys_key_hash"`
	UserID     string     `gorm:"column:user_id;not null;index:idx_api_keys_user_id"`
	ExpiresAt  *time.Time `gorm:"column:expires_at"`
	LastUsedAt *time.Time `gorm:"column:last_used_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;not null"`
}

func (APIKeyRecord) TableName() string {
	return "api_keys"
}
