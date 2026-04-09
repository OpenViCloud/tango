package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrServerNotFound = errors.New("server not found")
	ErrServerConflict = errors.New("server already exists")
)

type ServerStatus string

const (
	ServerStatusPending   ServerStatus = "pending"
	ServerStatusConnected ServerStatus = "connected"
	ServerStatusError     ServerStatus = "error"
)

type Server struct {
	ID         string
	Name       string
	PublicIP   string
	PrivateIP  string // optional; used as node_ip if set, otherwise PublicIP
	SSHUser    string // default: root
	SSHPort    int    // default: 22
	Status     ServerStatus
	ErrorMsg   string
	LastPingAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NodeIP returns the IP to use for intra-cluster communication.
func (s *Server) NodeIP() string {
	if s.PrivateIP != "" {
		return s.PrivateIP
	}
	return s.PublicIP
}

type ServerRepository interface {
	Save(ctx context.Context, s *Server) (*Server, error)
	Update(ctx context.Context, s *Server) (*Server, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Server, error)
	ListAll(ctx context.Context) ([]*Server, error)
}
