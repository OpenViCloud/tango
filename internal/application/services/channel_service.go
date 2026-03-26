package services

import (
	"context"
	"encoding/json"

	"tango/internal/contract/common"
)

type CreateChannelInput struct {
	Name        string
	Kind        string
	Status      string
	Credentials json.RawMessage
	Settings    json.RawMessage
}

type UpdateChannelInput struct {
	ID                 string
	Name               string
	Kind               string
	Status             string
	Credentials        json.RawMessage
	ReplaceCredentials bool
	Settings           json.RawMessage
}

type ChannelView struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Kind           string          `json:"kind"`
	Status         string          `json:"status"`
	HasCredentials bool            `json:"has_credentials"`
	Settings       json.RawMessage `json:"settings"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

type ChannelListView struct {
	Items      []ChannelView `json:"items"`
	PageIndex  int           `json:"pageIndex"`
	PageSize   int           `json:"pageSize"`
	TotalItems int64         `json:"totalItems"`
	TotalPage  int           `json:"totalPage"`
}

type TestChannelConnectionInput struct {
	Kind        string
	Credentials json.RawMessage
	Settings    json.RawMessage
}

type TestChannelConnectionView struct {
	Kind    string         `json:"kind"`
	OK      bool           `json:"ok"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type ChannelConnectionError struct {
	Code    string
	Message string
	Cause   error
}

func (e *ChannelConnectionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func (e *ChannelConnectionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type ChannelService interface {
	Create(ctx context.Context, input CreateChannelInput) (*ChannelView, error)
	Update(ctx context.Context, input UpdateChannelInput) (*ChannelView, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*ChannelView, error)
	List(ctx context.Context, req common.BaseRequestModel) (*ChannelListView, error)
	TestConnection(ctx context.Context, input TestChannelConnectionInput) (*TestChannelConnectionView, error)
}
