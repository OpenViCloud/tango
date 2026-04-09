package models

import "time"

type ServerRecord struct {
	ID          string     `gorm:"primaryKey;type:text"`
	Name        string     `gorm:"column:name;not null"`
	PublicIP    string     `gorm:"column:public_ip;not null"`
	PrivateIP   string     `gorm:"column:private_ip;not null;default:''"`
	SSHUser     string     `gorm:"column:ssh_user;not null;default:'root'"`
	SSHPort     int        `gorm:"column:ssh_port;not null;default:22"`
	Status      string     `gorm:"column:status;not null;default:'pending'"`
	ErrorMsg    string     `gorm:"column:error_msg;not null;default:''"`
	LastPingAt  *time.Time `gorm:"column:last_ping_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null"`
}

func (ServerRecord) TableName() string { return "servers" }
