package models

import "time"

type UserRoleRecord struct {
	UserID    string    `gorm:"column:user_id;primaryKey"`
	RoleID    string    `gorm:"column:role_id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (UserRoleRecord) TableName() string {
	return "user_roles"
}
