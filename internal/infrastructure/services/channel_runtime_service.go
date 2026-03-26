package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type channelRuntimeService struct {
	repo            domain.ChannelRepository
	cipher          appservices.SecretCipher
	discordRuntime  appservices.DiscordRuntimeService
	slackRuntime    appservices.SlackRuntimeService
	telegramRuntime appservices.TelegramRuntimeService
	whatsAppRuntime appservices.WhatsAppRuntimeService
}

type discordChannelCredentials struct {
	Token string `json:"token"`
}

type discordChannelSettings struct {
	RequireMention             bool     `json:"require_mention"`
	EnableTyping               bool     `json:"enable_typing"`
	EnableMessageContentIntent bool     `json:"enable_message_content_intent"`
	AllowedUserIDs             []string `json:"allowed_user_ids"`
}

type telegramChannelCredentials struct {
	Token string `json:"token"`
}

type slackChannelCredentials struct {
	BotToken string `json:"bot_token"`
	AppToken string `json:"app_token"`
}

type telegramChannelSettings struct {
	EnableTyping   bool     `json:"enable_typing"`
	AllowedUserIDs []string `json:"allowed_user_ids"`
}

type slackChannelSettings struct {
	RequireMention bool     `json:"require_mention"`
	EnableTyping   bool     `json:"enable_typing"`
	AllowedUserIDs []string `json:"allowed_user_ids"`
}

type whatsAppChannelSettings struct {
	AllowedUserIDs []string `json:"allowed_user_ids"`
}

func NewChannelRuntimeService(
	repo domain.ChannelRepository,
	cipher appservices.SecretCipher,
	discordRuntime appservices.DiscordRuntimeService,
	slackRuntime appservices.SlackRuntimeService,
	telegramRuntime appservices.TelegramRuntimeService,
	whatsAppRuntime appservices.WhatsAppRuntimeService,
) appservices.ChannelRuntimeService {
	return &channelRuntimeService{
		repo:            repo,
		cipher:          cipher,
		discordRuntime:  discordRuntime,
		slackRuntime:    slackRuntime,
		telegramRuntime: telegramRuntime,
		whatsAppRuntime: whatsAppRuntime,
	}
}

func (s *channelRuntimeService) Start(ctx context.Context, id string) (*appservices.ChannelRuntimeView, error) {
	channel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.verifyStoredChannelConnection(ctx, channel); err != nil {
		return nil, err
	}

	if err := s.startChannel(ctx, channel); err != nil {
		return nil, err
	}

	channel.Status = domain.ChannelStatusActive
	channel.UpdatedAt = time.Now().UTC()
	if _, err := s.repo.Update(ctx, channel); err != nil {
		return nil, err
	}
	if err := s.repo.SetStatusByKindExcept(ctx, channel.Kind, channel.ID, domain.ChannelStatusDisabled); err != nil {
		return nil, err
	}

	return s.viewFor(ctx, channel), nil
}

func (s *channelRuntimeService) Stop(ctx context.Context, id string) (*appservices.ChannelRuntimeView, error) {
	channel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.stopKind(ctx, channel.Kind); err != nil {
		return nil, err
	}

	channel.Status = domain.ChannelStatusDisabled
	channel.UpdatedAt = time.Now().UTC()
	if _, err := s.repo.Update(ctx, channel); err != nil {
		return nil, err
	}

	return s.viewFor(ctx, channel), nil
}

func (s *channelRuntimeService) Restart(ctx context.Context, id string) (*appservices.ChannelRuntimeView, error) {
	channel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.verifyStoredChannelConnection(ctx, channel); err != nil {
		return nil, err
	}

	if err := s.stopKind(ctx, channel.Kind); err != nil {
		return nil, err
	}
	if err := s.startChannel(ctx, channel); err != nil {
		return nil, err
	}

	channel.Status = domain.ChannelStatusActive
	channel.UpdatedAt = time.Now().UTC()
	if _, err := s.repo.Update(ctx, channel); err != nil {
		return nil, err
	}
	if err := s.repo.SetStatusByKindExcept(ctx, channel.Kind, channel.ID, domain.ChannelStatusDisabled); err != nil {
		return nil, err
	}

	return s.viewFor(ctx, channel), nil
}

func (s *channelRuntimeService) StartActiveChannels(ctx context.Context) error {
	channels, err := s.repo.ListActiveConfigured(ctx)
	if err != nil {
		return err
	}
	for _, channel := range channels {
		if err := s.startChannel(ctx, channel); err != nil {
			return fmt.Errorf("start channel %s: %w", channel.Name, err)
		}
		if err := s.repo.SetStatusByKindExcept(ctx, channel.Kind, channel.ID, domain.ChannelStatusDisabled); err != nil {
			return err
		}
	}
	return nil
}

func (s *channelRuntimeService) GetQRCode(ctx context.Context, id string) (string, error) {
	channel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	switch channel.Kind {
	case domain.ChannelKindWhatsApp:
		return s.whatsAppRuntime.QRCode(ctx), nil
	case domain.ChannelKindDiscord, domain.ChannelKindTelegram, domain.ChannelKindSlack:
		return "", domain.ErrUnsupportedChannelKind
	default:
		return "", domain.ErrUnsupportedChannelKind
	}
}

func (s *channelRuntimeService) startChannel(ctx context.Context, channel *domain.Channel) error {
	switch channel.Kind {
	case domain.ChannelKindDiscord:
		cfg, err := buildDiscordRuntimeConfig(ctx, s.cipher, channel)
		if err != nil {
			return err
		}
		return s.discordRuntime.Start(ctx, cfg)
	case domain.ChannelKindSlack:
		cfg, err := buildSlackRuntimeConfig(ctx, s.cipher, channel)
		if err != nil {
			return err
		}
		return s.slackRuntime.Start(ctx, cfg)
	case domain.ChannelKindTelegram:
		cfg, err := buildTelegramRuntimeConfig(ctx, s.cipher, channel)
		if err != nil {
			return err
		}
		return s.telegramRuntime.Start(ctx, cfg)
	case domain.ChannelKindWhatsApp:
		cfg, err := buildWhatsAppRuntimeConfig(channel)
		if err != nil {
			return err
		}
		return s.whatsAppRuntime.Start(ctx, cfg)
	default:
		return domain.ErrUnsupportedChannelKind
	}
}

func (s *channelRuntimeService) stopKind(ctx context.Context, kind domain.ChannelKind) error {
	switch kind {
	case domain.ChannelKindDiscord:
		return s.discordRuntime.Stop(ctx)
	case domain.ChannelKindSlack:
		return s.slackRuntime.Stop(ctx)
	case domain.ChannelKindTelegram:
		return s.telegramRuntime.Stop(ctx)
	case domain.ChannelKindWhatsApp:
		return s.whatsAppRuntime.Stop(ctx)
	default:
		return domain.ErrUnsupportedChannelKind
	}
}

func (s *channelRuntimeService) viewFor(ctx context.Context, channel *domain.Channel) *appservices.ChannelRuntimeView {
	running := false
	switch channel.Kind {
	case domain.ChannelKindDiscord:
		running = s.discordRuntime.Status(ctx).Running
	case domain.ChannelKindSlack:
		running = s.slackRuntime.Status(ctx).Running
	case domain.ChannelKindTelegram:
		running = s.telegramRuntime.Status(ctx).Running
	case domain.ChannelKindWhatsApp:
		running = s.whatsAppRuntime.Status(ctx).Running
	}

	return &appservices.ChannelRuntimeView{
		ID:             channel.ID,
		Name:           channel.Name,
		Kind:           string(channel.Kind),
		Status:         string(channel.Status),
		Running:        running,
		HasCredentials: strings.TrimSpace(channel.EncryptedCredentials) != "",
	}
}

func buildDiscordRuntimeConfig(ctx context.Context, cipher appservices.SecretCipher, channel *domain.Channel) (appservices.DiscordRuntimeConfig, error) {
	var creds discordChannelCredentials
	if err := decryptChannelCredentials(ctx, cipher, channel, &creds); err != nil {
		return appservices.DiscordRuntimeConfig{}, err
	}
	var settings discordChannelSettings
	if err := json.Unmarshal([]byte(channel.SettingsJSON), &settings); err != nil {
		return appservices.DiscordRuntimeConfig{}, fmt.Errorf("decode discord settings: %w", err)
	}
	return appservices.DiscordRuntimeConfig{
		ChannelID:                  channel.ID,
		Token:                      creds.Token,
		RequireMention:             settings.RequireMention,
		EnableTyping:               settings.EnableTyping,
		EnableMessageContentIntent: settings.EnableMessageContentIntent,
		AllowedUserIDs:             settings.AllowedUserIDs,
	}, nil
}

func buildTelegramRuntimeConfig(ctx context.Context, cipher appservices.SecretCipher, channel *domain.Channel) (appservices.TelegramRuntimeConfig, error) {
	var creds telegramChannelCredentials
	if err := decryptChannelCredentials(ctx, cipher, channel, &creds); err != nil {
		return appservices.TelegramRuntimeConfig{}, err
	}
	var settings telegramChannelSettings
	if err := json.Unmarshal([]byte(channel.SettingsJSON), &settings); err != nil {
		return appservices.TelegramRuntimeConfig{}, fmt.Errorf("decode telegram settings: %w", err)
	}
	return appservices.TelegramRuntimeConfig{
		ChannelID:      channel.ID,
		Token:          creds.Token,
		EnableTyping:   settings.EnableTyping,
		AllowedUserIDs: settings.AllowedUserIDs,
	}, nil
}

func buildSlackRuntimeConfig(ctx context.Context, cipher appservices.SecretCipher, channel *domain.Channel) (appservices.SlackRuntimeConfig, error) {
	var creds slackChannelCredentials
	if err := decryptChannelCredentials(ctx, cipher, channel, &creds); err != nil {
		return appservices.SlackRuntimeConfig{}, err
	}
	var settings slackChannelSettings
	if err := json.Unmarshal([]byte(channel.SettingsJSON), &settings); err != nil {
		return appservices.SlackRuntimeConfig{}, fmt.Errorf("decode slack settings: %w", err)
	}
	return appservices.SlackRuntimeConfig{
		ChannelID:      channel.ID,
		BotToken:       creds.BotToken,
		AppToken:       creds.AppToken,
		RequireMention: settings.RequireMention,
		EnableTyping:   settings.EnableTyping,
		AllowedUserIDs: settings.AllowedUserIDs,
	}, nil
}

func buildWhatsAppRuntimeConfig(channel *domain.Channel) (appservices.WhatsAppRuntimeConfig, error) {
	if channel == nil {
		return appservices.WhatsAppRuntimeConfig{}, fmt.Errorf("channel is nil")
	}

	var settings whatsAppChannelSettings
	if err := json.Unmarshal([]byte(channel.SettingsJSON), &settings); err != nil {
		return appservices.WhatsAppRuntimeConfig{}, fmt.Errorf("decode whatsapp settings: %w", err)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return appservices.WhatsAppRuntimeConfig{}, fmt.Errorf("resolve user config dir: %w", err)
	}

	return appservices.WhatsAppRuntimeConfig{
		ChannelID:      channel.ID,
		SessionPath:    filepath.Join(configDir, "tango", "whatsapp", channel.ID+".db"),
		AllowedUserIDs: settings.AllowedUserIDs,
	}, nil
}

func (s *channelRuntimeService) verifyStoredChannelConnection(ctx context.Context, channel *domain.Channel) error {
	if channel == nil {
		return domain.ErrInvalidInput
	}
	if channel.Kind == domain.ChannelKindWhatsApp || channel.Kind == domain.ChannelKindWeb {
		return nil
	}
	if strings.TrimSpace(channel.EncryptedCredentials) == "" {
		return domain.ErrInvalidInput
	}
	decrypted, err := s.cipher.Decrypt(ctx, channel.EncryptedCredentials)
	if err != nil {
		return domain.ErrChannelEncryptionFailed
	}
	return verifyChannelConnectionByKind(ctx, string(channel.Kind), json.RawMessage(decrypted))
}

func decryptChannelCredentials(ctx context.Context, cipher appservices.SecretCipher, channel *domain.Channel, out any) error {
	if channel == nil {
		return fmt.Errorf("channel is nil")
	}
	if strings.TrimSpace(channel.EncryptedCredentials) == "" {
		return fmt.Errorf("channel credentials are empty")
	}
	raw, err := cipher.Decrypt(ctx, channel.EncryptedCredentials)
	if err != nil {
		return fmt.Errorf("decrypt channel credentials: %w", err)
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return fmt.Errorf("decode channel credentials: %w", err)
	}
	return nil
}
