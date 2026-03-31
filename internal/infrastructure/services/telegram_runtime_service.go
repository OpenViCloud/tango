package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"

	appservices "tango/internal/application/services"
	telegramchannel "tango/internal/channels/telegram"
	"tango/internal/messaging/inbound"
)

type telegramRuntimeService struct {
	rootCtx   context.Context
	publisher inbound.Publisher
	logger    *slog.Logger
	navigator appservices.TelegramProjectNavigator

	mu      sync.Mutex
	channel *telegramchannel.Channel
	cfg     appservices.TelegramRuntimeConfig
}

func NewTelegramRuntimeService(rootCtx context.Context, publisher inbound.Publisher, logger *slog.Logger, navigator appservices.TelegramProjectNavigator) appservices.TelegramRuntimeService {
	if logger == nil {
		logger = slog.Default()
	}
	if rootCtx == nil {
		rootCtx = context.Background()
	}
	return &telegramRuntimeService{
		rootCtx:   rootCtx,
		publisher: publisher,
		logger:    logger,
		navigator: navigator,
	}
}

func (s *telegramRuntimeService) Status(_ context.Context) appservices.TelegramRuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return appservices.TelegramRuntimeStatus{
		Running:         s.channel != nil,
		TokenConfigured: strings.TrimSpace(s.cfg.Token) != "",
		EnableTyping:    s.cfg.EnableTyping,
		AllowedUserIDs:  append([]string(nil), s.cfg.AllowedUserIDs...),
	}
}

func (s *telegramRuntimeService) Start(ctx context.Context, cfg appservices.TelegramRuntimeConfig) error {
	normalized := normalizeTelegramRuntimeConfig(cfg)
	if strings.TrimSpace(normalized.Token) == "" {
		return errors.New("telegram token is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.stopLocked(ctx); err != nil {
		return err
	}

	ch, err := telegramchannel.New(telegramchannel.Config{
		ChannelID:      normalized.ChannelID,
		Token:          normalized.Token,
		AllowedUserIDs: idsToSet(normalized.AllowedUserIDs),
		EnableTyping:   normalized.EnableTyping,
		Navigator:      s.navigator,
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

func (s *telegramRuntimeService) Restart(ctx context.Context) error {
	s.mu.Lock()
	cfg := s.cfg
	s.mu.Unlock()

	if strings.TrimSpace(cfg.Token) == "" {
		return errors.New("telegram token is not configured")
	}
	return s.Start(ctx, cfg)
}

func (s *telegramRuntimeService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked(ctx)
}

func (s *telegramRuntimeService) stopLocked(_ context.Context) error {
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

func normalizeTelegramRuntimeConfig(cfg appservices.TelegramRuntimeConfig) appservices.TelegramRuntimeConfig {
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
