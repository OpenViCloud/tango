package models

import "time"

type ClusterRecord struct {
	ID             string    `gorm:"primaryKey;type:text"`
	Name           string    `gorm:"column:name;not null"`
	Status         string    `gorm:"column:status;not null;default:'pending'"`
	ErrorMsg       string    `gorm:"column:error_msg;not null;default:''"`
	K8sVersion     string    `gorm:"column:k8s_version;not null;default:'v1.30'"`
	PodCIDR        string    `gorm:"column:pod_cidr;not null;default:'192.168.0.0/16'"`
	KubeconfigEnc  string    `gorm:"column:kubeconfig_enc;not null;default:''"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`

	Nodes []ClusterNodeRecord `gorm:"foreignKey:ClusterID"`
}

func (ClusterRecord) TableName() string { return "clusters" }

type ClusterNodeRecord struct {
	ClusterID string `gorm:"column:cluster_id;primaryKey;type:text"`
	ServerID  string `gorm:"column:server_id;primaryKey;type:text"`
	Role      string `gorm:"column:role;not null"`
}

func (ClusterNodeRecord) TableName() string { return "cluster_nodes" }
