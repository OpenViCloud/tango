package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"

	appservices "tango/internal/application/services"
	slackchannel "tango/internal/channels/slack"
	"tango/internal/messaging/inbound"
)

type slackRuntimeService struct {
	rootCtx   context.Context
	publisher inbound.Publisher
	logger    *slog.Logger

	mu      sync.Mutex
	channel *slackchannel.Channel
	cfg     appservices.SlackRuntimeConfig
}

func NewSlackRuntimeService(rootCtx context.Context, publisher inbound.Publisher, logger *slog.Logger) appservices.SlackRuntimeService {
	if logger == nil {
		logger = slog.Default()
	}
	if rootCtx == nil {
		rootCtx = context.Background()
	}
	return &slackRuntimeService{
		rootCtx:   rootCtx,
		publisher: publisher,
		logger:    logger,
	}
}

func (s *slackRuntimeService) Status(_ context.Context) appservices.SlackRuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return appservices.SlackRuntimeStatus{
		Running:        s.channel != nil,
		BotConfigured:  strings.TrimSpace(s.cfg.BotToken) != "",
		AppConfigured:  strings.TrimSpace(s.cfg.AppToken) != "",
		RequireMention: s.cfg.RequireMention,
		EnableTyping:   s.cfg.EnableTyping,
		AllowedUserIDs: append([]string(nil), s.cfg.AllowedUserIDs...),
	}
}

func (s *slackRuntimeService) Start(ctx context.Context, cfg appservices.SlackRuntimeConfig) error {
	normalized := normalizeSlackRuntimeConfig(cfg)
	if strings.TrimSpace(normalized.BotToken) == "" {
		return errors.New("slack bot token is required")
	}
	if strings.TrimSpace(normalized.AppToken) == "" {
		return errors.New("slack app token is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.stopLocked(ctx); err != nil {
		return err
	}

	ch, err := slackchannel.New(slackchannel.Config{
		ChannelID:      normalized.ChannelID,
		BotToken:       normalized.BotToken,
		AppToken:       normalized.AppToken,
		AllowedUserIDs: idsToSet(normalized.AllowedUserIDs),
		RequireMention: normalized.RequireMention,
		EnableTyping:   normalized.EnableTyping,
	}, s.publisher, s.logger)
	if err != nil {
		return err
	}

	if aware, ok := s.publisher.(inbound.SenderAwarePublisher); ok {
		aware.SetSender(ch)
	}

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

func (s *slackRuntimeService) Restart(ctx context.Context) error {
	s.mu.Lock()
	cfg := s.cfg
	s.mu.Unlock()

	if strings.TrimSpace(cfg.BotToken) == "" || strings.TrimSpace(cfg.AppToken) == "" {
		return errors.New("slack runtime is not configured")
	}
	return s.Start(ctx, cfg)
}

func (s *slackRuntimeService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked(ctx)
}

func (s *slackRuntimeService) stopLocked(_ context.Context) error {
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

func normalizeSlackRuntimeConfig(cfg appservices.SlackRuntimeConfig) appservices.SlackRuntimeConfig {
	cfg.BotToken = strings.TrimSpace(cfg.BotToken)
	cfg.AppToken = strings.TrimSpace(cfg.AppToken)

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
