package domain

import "context"

type RoleRepository interface {
	EnsureRole(ctx context.Context, role *Role) error
	Save(ctx context.Context, role *Role) (*Role, error)
	Update(ctx context.Context, role *Role) (*Role, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Role, error)
	GetByName(ctx context.Context, name string) (*Role, error)
	GetAll(ctx context.Context, opts RoleListOptions) (*RoleListResult, error)
	ListByUserID(ctx context.Context, userID string) ([]*Role, error)
	AssignRoleToUser(ctx context.Context, userID, roleID string) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID string) error
}
