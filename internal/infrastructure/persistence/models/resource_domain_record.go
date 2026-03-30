package models

import "time"

type ResourceDomainRecord struct {
	ID         string         `gorm:"primaryKey;type:text"`
	ResourceID string         `gorm:"column:resource_id;type:text;not null;index"`
	Host       string         `gorm:"column:host;type:text;not null;uniqueIndex"`
	TLSEnabled bool           `gorm:"column:tls_enabled;not null;default:false"`
	Type       string         `gorm:"column:type;type:varchar(16);not null;default:'custom'"`
	Verified   bool           `gorm:"column:verified;not null;default:false"`
	VerifiedAt *time.Time     `gorm:"column:verified_at"`
	CreatedAt  time.Time      `gorm:"column:created_at;not null"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;not null"`
	Resource   ResourceRecord `gorm:"foreignKey:ResourceID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ResourceDomainRecord) TableName() string { return "resource_domains" }
