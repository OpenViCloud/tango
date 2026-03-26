package services

import "context"

// DiscordRuntimeConfig is the app-level runtime configuration for the Discord channel.
type DiscordRuntimeConfig struct {
	ChannelID                  string
	Token                      string
	RequireMention             bool
	EnableTyping               bool
	EnableMessageContentIntent bool
	AllowedUserIDs             []string
}

// DiscordRuntimeStatus describes the live Discord runtime state exposed to callers.
type DiscordRuntimeStatus struct {
	Running                    bool
	TokenConfigured            bool
	RequireMention             bool
	EnableTyping               bool
	EnableMessageContentIntent bool
	AllowedUserIDs             []string
}

// DiscordRuntimeService manages the Discord channel lifecycle while the app is running.
type DiscordRuntimeService interface {
	Status(ctx context.Context) DiscordRuntimeStatus
	Start(ctx context.Context, cfg DiscordRuntimeConfig) error
	Restart(ctx context.Context) error
	Stop(ctx context.Context) error
}
