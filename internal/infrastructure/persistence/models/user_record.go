package models

import "time"

type UserRecord struct {
	ID           string     `gorm:"primaryKey;type:text"`
	Email        string     `gorm:"not null;uniqueIndex:idx_users_email"`
	Nickname     string     `gorm:"column:nickname"`
	FirstName    string     `gorm:"column:first_name;not null"`
	LastName     string     `gorm:"column:last_name;not null"`
	Phone        string     `gorm:"column:phone"`
	Address      string     `gorm:"column:address"`
	PasswordHash string     `gorm:"column:password_hash;not null"`
	Status       string     `gorm:"not null"`
	CreatedAt    time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;not null"`
	DeletedAt    *time.Time `gorm:"column:deleted_at"`
}

func (UserRecord) TableName() string {
	return "users"
}
