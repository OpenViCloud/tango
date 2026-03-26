package command

import (
	"context"

	"tango/internal/domain"
)

type AssignUserRoleCommand struct {
	UserID string
	RoleID string
}

type AssignUserRoleHandler struct {
	userRepo domain.UserRepository
	roleRepo domain.RoleRepository
}

func NewAssignUserRoleHandler(userRepo domain.UserRepository, roleRepo domain.RoleRepository) *AssignUserRoleHandler {
	return &AssignUserRoleHandler{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

func (h *AssignUserRoleHandler) Handle(ctx context.Context, cmd AssignUserRoleCommand) error {
	if _, err := h.userRepo.GetByID(ctx, cmd.UserID); err != nil {
		return err
	}
	if _, err := h.roleRepo.GetByID(ctx, cmd.RoleID); err != nil {
		return err
	}
	return h.roleRepo.AssignRoleToUser(ctx, cmd.UserID, cmd.RoleID)
}

type RemoveUserRoleCommand struct {
	UserID string
	RoleID string
}

type RemoveUserRoleHandler struct {
	userRepo domain.UserRepository
	roleRepo domain.RoleRepository
}

func NewRemoveUserRoleHandler(userRepo domain.UserRepository, roleRepo domain.RoleRepository) *RemoveUserRoleHandler {
	return &RemoveUserRoleHandler{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

func (h *RemoveUserRoleHandler) Handle(ctx context.Context, cmd RemoveUserRoleCommand) error {
	if _, err := h.userRepo.GetByID(ctx, cmd.UserID); err != nil {
		return err
	}
	if _, err := h.roleRepo.GetByID(ctx, cmd.RoleID); err != nil {
		return err
	}
	return h.roleRepo.RemoveRoleFromUser(ctx, cmd.UserID, cmd.RoleID)
}
