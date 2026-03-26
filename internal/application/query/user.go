package query

import (
	"context"
	"fmt"

	"tango/internal/contract/common"
	"tango/internal/domain"
)

type StatusView struct {
	Version string `json:"version"`
	Status  string `json:"status"`
	Uptime  string `json:"uptime"`
}

type GetStatusHandler struct{}

func NewGetStatusHandler() *GetStatusHandler {
	return &GetStatusHandler{}
}

func (h *GetStatusHandler) Handle(context.Context) *StatusView {
	return &StatusView{
		Version: "0.1.0",
		Status:  "ok",
		Uptime:  "0s",
	}
}

type GetUserByIDQuery struct {
	ID string
}

type GetUserByIDHandler struct {
	repo domain.UserRepository
}

func NewGetUserByIDHandler(repo domain.UserRepository) *GetUserByIDHandler {
	return &GetUserByIDHandler{repo: repo}
}

func (h *GetUserByIDHandler) Handle(ctx context.Context, q GetUserByIDQuery) (*domain.User, error) {
	user, err := h.repo.GetByID(ctx, q.ID)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

type ListUsersQuery struct {
	common.BaseRequestModel
}

type ListUsersHandler struct {
	repo domain.UserRepository
}

func NewListUsersHandler(repo domain.UserRepository) *ListUsersHandler {
	return &ListUsersHandler{repo: repo}
}

func (h *ListUsersHandler) Handle(ctx context.Context, q ListUsersQuery) (*domain.UserListResult, error) {
	users, err := h.repo.GetAll(ctx, domain.UserListOptions{
		PageIndex:  q.PageIndex,
		PageSize:   q.PageSize,
		SearchText: q.SearchText,
		OrderBy:    q.OrderBy,
		Ascending:  q.Ascending,
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}
