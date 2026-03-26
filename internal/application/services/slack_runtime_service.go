package services

import "context"

type SlackRuntimeConfig struct {
	ChannelID      string
	BotToken       string
	AppToken       string
	RequireMention bool
	EnableTyping   bool
	AllowedUserIDs []string
}

type SlackRuntimeStatus struct {
	Running        bool     `json:"running"`
	BotConfigured  bool     `json:"bot_configured"`
	AppConfigured  bool     `json:"app_configured"`
	RequireMention bool     `json:"require_mention"`
	EnableTyping   bool     `json:"enable_typing"`
	AllowedUserIDs []string `json:"allowed_user_ids"`
}

type SlackRuntimeService interface {
	Status(ctx context.Context) SlackRuntimeStatus
	Start(ctx context.Context, cfg SlackRuntimeConfig) error
	Restart(ctx context.Context) error
	Stop(ctx context.Context) error
}
