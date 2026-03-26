package discord

import (
	"context"
	"log/slog"

	"tango/internal/config"
	"tango/internal/messaging/inbound"
)

// Bootstrap creates and starts the Discord channel from app config.
// It returns nil when Discord is not configured.
func Bootstrap(ctx context.Context, cfg *config.Config, publisher inbound.Publisher, logger *slog.Logger) (*DiscordChannel, error) {
	if cfg == nil || cfg.DiscordToken == "" {
		if logger != nil {
			logger.Info("discord channel disabled", "reason", "DISCORD_BOT_TOKEN is empty")
		}
		return nil, nil
	}

	ch, err := New(
		Config{
			Token:                      cfg.DiscordToken,
			AllowedUserIDs:             cfg.DiscordAllowedUserIDs,
			RequireMention:             cfg.DiscordRequireMention,
			EnableTyping:               cfg.DiscordEnableTyping,
			EnableMessageContentIntent: cfg.DiscordEnableMessageContentIntent,
		},
		publisher,
		logger,
	)
	if err != nil {
		return nil, err
	}

	if aware, ok := publisher.(inbound.SenderAwarePublisher); ok {
		aware.SetSender(ch)
	}

	if err := ch.Start(ctx); err != nil {
		return nil, err
	}

	if logger != nil {
		logger.Info("discord channel enabled")
	}

	return ch, nil
}
