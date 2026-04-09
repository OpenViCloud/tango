package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrClusterNotFound = errors.New("cluster not found")
)

type ClusterStatus string

const (
	ClusterStatusPending      ClusterStatus = "pending"
	ClusterStatusProvisioning ClusterStatus = "provisioning"
	ClusterStatusReady        ClusterStatus = "ready"
	ClusterStatusError        ClusterStatus = "error"
)

type ClusterNodeRole string

const (
	ClusterNodeRoleMaster ClusterNodeRole = "master"
	ClusterNodeRoleWorker ClusterNodeRole = "worker"
)

type ClusterNode struct {
	ServerID string
	Role     ClusterNodeRole
}

type Cluster struct {
	ID         string
	Name       string
	Status     ClusterStatus
	ErrorMsg   string
	Nodes      []ClusterNode
	K8sVersion string // e.g. "v1.30"
	PodCIDR    string // e.g. "192.168.0.0/16"
	Kubeconfig string // fetched after provisioning; stored encrypted
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ClusterRepository interface {
	Save(ctx context.Context, c *Cluster) (*Cluster, error)
	Update(ctx context.Context, c *Cluster) (*Cluster, error)
	UpdateStatus(ctx context.Context, id string, status ClusterStatus, errMsg string) error
	UpdateKubeconfig(ctx context.Context, id string, kubeconfigEnc string) error
	GetByID(ctx context.Context, id string) (*Cluster, error)
	ListAll(ctx context.Context) ([]*Cluster, error)
	Delete(ctx context.Context, id string) error
}
