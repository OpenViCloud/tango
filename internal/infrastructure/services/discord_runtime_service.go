package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"

	appservices "tango/internal/application/services"
	discordchannel "tango/internal/channels/discord"
	"tango/internal/messaging/inbound"
)

type discordRuntimeService struct {
	rootCtx   context.Context
	publisher inbound.Publisher
	logger    *slog.Logger

	mu      sync.Mutex
	channel *discordchannel.DiscordChannel
	cfg     appservices.DiscordRuntimeConfig
}

// NewDiscordRuntimeService constructs a runtime Discord channel manager.
func NewDiscordRuntimeService(rootCtx context.Context, publisher inbound.Publisher, logger *slog.Logger) appservices.DiscordRuntimeService {
	if logger == nil {
		logger = slog.Default()
	}
	if rootCtx == nil {
		rootCtx = context.Background()
	}

	return &discordRuntimeService{
		rootCtx:   rootCtx,
		publisher: publisher,
		logger:    logger,
	}
}

func (s *discordRuntimeService) Status(_ context.Context) appservices.DiscordRuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return appservices.DiscordRuntimeStatus{
		Running:                    s.channel != nil,
		TokenConfigured:            strings.TrimSpace(s.cfg.Token) != "",
		RequireMention:             s.cfg.RequireMention,
		EnableTyping:               s.cfg.EnableTyping,
		EnableMessageContentIntent: s.cfg.EnableMessageContentIntent,
		AllowedUserIDs:             append([]string(nil), s.cfg.AllowedUserIDs...),
	}
}

func (s *discordRuntimeService) Start(ctx context.Context, cfg appservices.DiscordRuntimeConfig) error {
	normalized := normalizeDiscordRuntimeConfig(cfg)
	if strings.TrimSpace(normalized.Token) == "" {
		return errors.New("discord token is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.stopLocked(ctx); err != nil {
		return err
	}

	ch, err := discordchannel.New(discordchannel.Config{
		ChannelID:                  normalized.ChannelID,
		Token:                      normalized.Token,
		AllowedUserIDs:             idsToSet(normalized.AllowedUserIDs),
		RequireMention:             normalized.RequireMention,
		EnableTyping:               normalized.EnableTyping,
		EnableMessageContentIntent: normalized.EnableMessageContentIntent,
	}, s.publisher, s.logger)
	if err != nil {
		return err
	}

	if aware, ok := s.publisher.(inbound.SenderAwarePublisher); ok {
		aware.SetSender(ch)
	}

	// The channel lifecycle must follow the app runtime, not a short-lived HTTP request context.
	if err := ch.Start(s.rootCtx); err != nil {
		if aware, ok := s.publisher.(inbound.SenderAwarePublisher); ok {
			aware.SetSender(nil)
		}
		return err
	}

	s.channel = ch
	s.cfg = normalized
	return nil
}

func (s *discordRuntimeService) Restart(ctx context.Context) error {
	s.mu.Lock()
	cfg := s.cfg
	s.mu.Unlock()

	if strings.TrimSpace(cfg.Token) == "" {
		return errors.New("discord token is not configured")
	}

	return s.Start(ctx, cfg)
}

func (s *discordRuntimeService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked(ctx)
}

func (s *discordRuntimeService) stopLocked(_ context.Context) error {
	if aware, ok := s.publisher.(inbound.SenderAwarePublisher); ok {
		aware.SetSender(nil)
	}

	if s.channel == nil {
		return nil
	}

	err := s.channel.Stop()
	s.channel = nil
	return err
}

func normalizeDiscordRuntimeConfig(cfg appservices.DiscordRuntimeConfig) appservices.DiscordRuntimeConfig {
	cfg.Token = strings.TrimSpace(cfg.Token)

	allowed := make([]string, 0, len(cfg.AllowedUserIDs))
	seen := make(map[string]struct{}, len(cfg.AllowedUserIDs))
	for _, raw := range cfg.AllowedUserIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		allowed = append(allowed, id)
	}
	cfg.AllowedUserIDs = allowed
	return cfg
}

func idsToSet(ids []string) map[string]bool {
	if len(ids) == 0 {
		return nil
	}

	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out[id] = true
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
