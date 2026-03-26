package services

import "context"

type ChannelRuntimeView struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	Running        bool   `json:"running"`
	HasCredentials bool   `json:"has_credentials"`
}

type ChannelRuntimeService interface {
	Start(ctx context.Context, id string) (*ChannelRuntimeView, error)
	Stop(ctx context.Context, id string) (*ChannelRuntimeView, error)
	Restart(ctx context.Context, id string) (*ChannelRuntimeView, error)
	StartActiveChannels(ctx context.Context) error
	GetQRCode(ctx context.Context, id string) (string, error)
}
