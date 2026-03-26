package services

import (
	"context"

	"tango/internal/contract/common"
)

type CreateRoleInput struct {
	Name        string
	Description string
}

type UpdateRoleInput struct {
	ID          string
	Name        string
	Description string
}

type RoleView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type RoleListView struct {
	Items      []RoleView `json:"items"`
	PageIndex  int        `json:"pageIndex"`
	PageSize   int        `json:"pageSize"`
	TotalItems int64      `json:"totalItems"`
	TotalPage  int        `json:"totalPage"`
}

type RoleService interface {
	Create(ctx context.Context, input CreateRoleInput) (*RoleView, error)
	Update(ctx context.Context, input UpdateRoleInput) (*RoleView, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*RoleView, error)
	List(ctx context.Context, req common.BaseRequestModel) (*RoleListView, error)
}
