package models

import "time"

type RoleRecord struct {
	ID          string `gorm:"primaryKey;type:text"`
	Name        string `gorm:"not null;uniqueIndex:idx_roles_name"`
	Description string
	IsSystem    bool      `gorm:"column:is_system;not null;default:false"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null"`
}

func (RoleRecord) TableName() string {
	return "roles"
}
