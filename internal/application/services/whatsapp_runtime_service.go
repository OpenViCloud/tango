package services

import "context"

type WhatsAppRuntimeConfig struct {
	ChannelID      string
	SessionPath    string
	AllowedUserIDs []string
}

type WhatsAppRuntimeStatus struct {
	Running        bool     `json:"running"`
	SessionPath    string   `json:"session_path"`
	AllowedUserIDs []string `json:"allowed_user_ids"`
}

type WhatsAppRuntimeService interface {
	Status(ctx context.Context) WhatsAppRuntimeStatus
	Start(ctx context.Context, cfg WhatsAppRuntimeConfig) error
	Restart(ctx context.Context) error
	Stop(ctx context.Context) error
	QRCode(ctx context.Context) string
}
