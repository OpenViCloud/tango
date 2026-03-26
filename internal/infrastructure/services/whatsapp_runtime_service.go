package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"

	appservices "tango/internal/application/services"
	whatsappchannel "tango/internal/channels/whatsapp"
	"tango/internal/messaging/inbound"
)

type whatsappRuntimeService struct {
	rootCtx   context.Context
	publisher inbound.Publisher
	logger    *slog.Logger

	mu      sync.Mutex
	channel *whatsappchannel.Channel
	cfg     appservices.WhatsAppRuntimeConfig
}

func NewWhatsAppRuntimeService(rootCtx context.Context, publisher inbound.Publisher, logger *slog.Logger) appservices.WhatsAppRuntimeService {
	if logger == nil {
		logger = slog.Default()
	}
	if rootCtx == nil {
		rootCtx = context.Background()
	}
	return &whatsappRuntimeService{
		rootCtx:   rootCtx,
		publisher: publisher,
		logger:    logger,
	}
}

func (s *whatsappRuntimeService) Status(_ context.Context) appservices.WhatsAppRuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return appservices.WhatsAppRuntimeStatus{
		Running:        s.channel != nil,
		SessionPath:    s.cfg.SessionPath,
		AllowedUserIDs: append([]string(nil), s.cfg.AllowedUserIDs...),
	}
}

func (s *whatsappRuntimeService) Start(ctx context.Context, cfg appservices.WhatsAppRuntimeConfig) error {
	normalized := normalizeWhatsAppRuntimeConfig(cfg)
	if strings.TrimSpace(normalized.SessionPath) == "" {
		return errors.New("whatsapp session path is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.stopLocked(ctx); err != nil {
		return err
	}

	ch, err := whatsappchannel.New(whatsappchannel.Config{
		ChannelID:      normalized.ChannelID,
		SessionPath:    normalized.SessionPath,
		AllowedUserIDs: idsToSet(normalized.AllowedUserIDs),
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

func (s *whatsappRuntimeService) Restart(ctx context.Context) error {
	s.mu.Lock()
	cfg := s.cfg
	s.mu.Unlock()

	if strings.TrimSpace(cfg.SessionPath) == "" {
		return errors.New("whatsapp session path is not configured")
	}
	return s.Start(ctx, cfg)
}

func (s *whatsappRuntimeService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked(ctx)
}

func (s *whatsappRuntimeService) QRCode(_ context.Context) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.channel == nil {
		return ""
	}
	return s.channel.QRCode()
}

func (s *whatsappRuntimeService) stopLocked(_ context.Context) error {
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

func normalizeWhatsAppRuntimeConfig(cfg appservices.WhatsAppRuntimeConfig) appservices.WhatsAppRuntimeConfig {
	cfg.SessionPath = strings.TrimSpace(cfg.SessionPath)

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
