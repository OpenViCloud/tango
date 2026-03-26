package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrRoleNotFound         = errors.New("role not found")
	ErrRoleAlreadyExists    = errors.New("role already exists")
	ErrSystemRoleProtected  = errors.New("system role is protected")
	ErrSystemRoleNameLocked = errors.New("system role name cannot be changed")
)

type RoleListOptions struct {
	PageIndex  int
	PageSize   int
	SearchText string
	OrderBy    string
	Ascending  bool
}

type RoleListResult struct {
	Items      []*Role
	TotalItems int64
}

type Role struct {
	ID          string
	Name        string
	Description string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewRole(id, name, description string, isSystem bool) (*Role, error) {
	now := time.Now().UTC()
	role := &Role{
		ID:          strings.TrimSpace(id),
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		IsSystem:    isSystem,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := role.Validate(); err != nil {
		return nil, err
	}
	return role, nil
}

func (r *Role) Validate() error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.Name) == "" {
		return ErrInvalidInput
	}
	r.Name = strings.TrimSpace(strings.ToLower(r.Name))
	r.Description = strings.TrimSpace(r.Description)
	return nil
}
