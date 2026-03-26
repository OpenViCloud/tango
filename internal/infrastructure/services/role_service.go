package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	appservices "tango/internal/application/services"
	"tango/internal/contract/common"
	"tango/internal/domain"
)

type roleService struct {
	repo domain.RoleRepository
}

func NewRoleService(repo domain.RoleRepository) appservices.RoleService {
	return &roleService{repo: repo}
}

func (s *roleService) Create(ctx context.Context, input appservices.CreateRoleInput) (*appservices.RoleView, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("role service is not initialized")
	}
	if existing, err := s.repo.GetByName(ctx, input.Name); err == nil && existing != nil {
		return nil, domain.ErrRoleAlreadyExists
	}

	role, err := domain.NewRole(newRoleID(), input.Name, input.Description, false)
	if err != nil {
		return nil, err
	}
	saved, err := s.repo.Save(ctx, role)
	if err != nil {
		return nil, err
	}
	return toRoleView(saved), nil
}

func (s *roleService) Update(ctx context.Context, input appservices.UpdateRoleInput) (*appservices.RoleView, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("role service is not initialized")
	}

	role, err := s.repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	nextName := strings.TrimSpace(strings.ToLower(input.Name))
	if role.IsSystem && nextName != "" && nextName != role.Name {
		return nil, domain.ErrSystemRoleNameLocked
	}
	if nextName != "" {
		if existing, err := s.repo.GetByName(ctx, nextName); err == nil && existing != nil && existing.ID != role.ID {
			return nil, domain.ErrRoleAlreadyExists
		}
		role.Name = nextName
	}
	role.Description = strings.TrimSpace(input.Description)
	role.UpdatedAt = time.Now().UTC()

	if err := role.Validate(); err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, role)
	if err != nil {
		return nil, err
	}
	return toRoleView(updated), nil
}

func (s *roleService) Delete(ctx context.Context, id string) error {
	if s.repo == nil {
		return fmt.Errorf("role service is not initialized")
	}
	role, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return domain.ErrSystemRoleProtected
	}
	return s.repo.Delete(ctx, id)
}

func (s *roleService) GetByID(ctx context.Context, id string) (*appservices.RoleView, error) {
	role, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toRoleView(role), nil
}

func (s *roleService) List(ctx context.Context, req common.BaseRequestModel) (*appservices.RoleListView, error) {
	result, err := s.repo.GetAll(ctx, domain.RoleListOptions{
		PageIndex:  req.PageIndex,
		PageSize:   req.PageSize,
		SearchText: req.SearchText,
		OrderBy:    req.OrderBy,
		Ascending:  req.Ascending,
	})
	if err != nil {
		return nil, err
	}

	items := make([]appservices.RoleView, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, *toRoleView(item))
	}

	totalPage := 0
	if req.PageSize > 0 {
		totalPage = int((result.TotalItems + int64(req.PageSize) - 1) / int64(req.PageSize))
	}

	return &appservices.RoleListView{
		Items:      items,
		PageIndex:  req.PageIndex,
		PageSize:   req.PageSize,
		TotalItems: result.TotalItems,
		TotalPage:  totalPage,
	}, nil
}

func toRoleView(role *domain.Role) *appservices.RoleView {
	if role == nil {
		return nil
	}
	return &appservices.RoleView{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		CreatedAt:   role.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   role.UpdatedAt.Format(time.RFC3339),
	}
}

func newRoleID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("role-%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
