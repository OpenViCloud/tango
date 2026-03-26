package services

import "context"

type TelegramRuntimeConfig struct {
	ChannelID      string
	Token          string
	EnableTyping   bool
	AllowedUserIDs []string
}

type TelegramRuntimeStatus struct {
	Running         bool     `json:"running"`
	TokenConfigured bool     `json:"token_configured"`
	EnableTyping    bool     `json:"enable_typing"`
	AllowedUserIDs  []string `json:"allowed_user_ids"`
}

type TelegramRuntimeService interface {
	Status(ctx context.Context) TelegramRuntimeStatus
	Start(ctx context.Context, cfg TelegramRuntimeConfig) error
	Restart(ctx context.Context) error
	Stop(ctx context.Context) error
}
