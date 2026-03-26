package telegram

import (
	"context"
	"log/slog"

	"tango/internal/config"
	"tango/internal/messaging/inbound"
)

// Bootstrap creates and starts the Telegram channel from app config.
// It returns nil when Telegram is not configured.
func Bootstrap(ctx context.Context, cfg *config.Config, publisher inbound.Publisher, logger *slog.Logger) (*Channel, error) {
	if cfg == nil || cfg.TelegramToken == "" {
		if logger != nil {
			logger.Info("telegram channel disabled", "reason", "TELEGRAM_BOT_TOKEN is empty")
		}
		return nil, nil
	}

	ch, err := New(Config{
		Token:          cfg.TelegramToken,
		AllowedUserIDs: cfg.TelegramAllowedUserIDs,
		EnableTyping:   cfg.TelegramEnableTyping,
	}, publisher, logger)
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
		logger.Info("telegram channel enabled")
	}

	return ch, nil
}
