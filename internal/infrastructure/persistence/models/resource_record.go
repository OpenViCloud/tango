package models

import "time"

type ResourceRecord struct {
	ID            string                 `gorm:"primaryKey;type:text"`
	Name          string                 `gorm:"not null"`
	Type          string                 `gorm:"column:type;not null"`
	Status        string                 `gorm:"column:status;not null"`
	ContainerID   string                 `gorm:"column:container_id;type:text"`
	Image         string                 `gorm:"column:image;not null"`
	Tag           string                 `gorm:"column:tag"`
	Config        string                 `gorm:"column:config;type:text"`
	EnvironmentID string                 `gorm:"column:environment_id;type:text;not null;index"`
	CreatedBy     string                 `gorm:"column:created_by;type:text"`
	CreatedAt     time.Time              `gorm:"column:created_at;not null"`
	UpdatedAt     time.Time              `gorm:"column:updated_at;not null"`
	Ports         []ResourcePortRecord   `gorm:"foreignKey:ResourceID;references:ID;constraint:OnDelete:CASCADE"`
	EnvVars       []ResourceEnvVarRecord `gorm:"foreignKey:ResourceID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ResourceRecord) TableName() string { return "resources" }

type ResourcePortRecord struct {
	ID           string         `gorm:"primaryKey;type:text"`
	ResourceID   string         `gorm:"column:resource_id;type:text;not null;index"`
	HostPort     int            `gorm:"column:host_port;not null;uniqueIndex"`
	InternalPort int            `gorm:"column:internal_port;not null"`
	Proto        string         `gorm:"column:proto;not null;default:'tcp'"`
	Label        string         `gorm:"column:label"`
	Resource     ResourceRecord `gorm:"foreignKey:ResourceID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ResourcePortRecord) TableName() string { return "resource_ports" }

type ResourceEnvVarRecord struct {
	ID         string         `gorm:"primaryKey;type:text"`
	ResourceID string         `gorm:"column:resource_id;type:text;not null;index"`
	Key        string         `gorm:"column:key;not null"`
	Value      string         `gorm:"column:value;not null;default:''"`
	IsSecret   bool           `gorm:"column:is_secret;not null;default:false"`
	CreatedAt  time.Time      `gorm:"column:created_at;not null"`
	Resource   ResourceRecord `gorm:"foreignKey:ResourceID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ResourceEnvVarRecord) TableName() string { return "resource_env_vars" }
