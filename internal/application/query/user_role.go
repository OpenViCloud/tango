package query

import (
	"context"
	"fmt"

	"tango/internal/domain"
)

type ListUserRolesQuery struct {
	UserID string
}

type ListUserRolesHandler struct {
	userRepo domain.UserRepository
	roleRepo domain.RoleRepository
}

func NewListUserRolesHandler(userRepo domain.UserRepository, roleRepo domain.RoleRepository) *ListUserRolesHandler {
	return &ListUserRolesHandler{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

func (h *ListUserRolesHandler) Handle(ctx context.Context, q ListUserRolesQuery) ([]*domain.Role, error) {
	if _, err := h.userRepo.GetByID(ctx, q.UserID); err != nil {
		return nil, err
	}
	items, err := h.roleRepo.ListByUserID(ctx, q.UserID)
	if err != nil {
		return nil, fmt.Errorf("list user roles: %w", err)
	}
	return items, nil
}
