package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appservices "tango/internal/application/services"
	"tango/internal/contract/common"
	"tango/internal/domain"
)

type channelService struct {
	repo   domain.ChannelRepository
	cipher appservices.SecretCipher
}

func NewChannelService(repo domain.ChannelRepository, cipher appservices.SecretCipher) appservices.ChannelService {
	return &channelService{repo: repo, cipher: cipher}
}

func (s *channelService) Create(ctx context.Context, input appservices.CreateChannelInput) (*appservices.ChannelView, error) {
	if s.repo == nil || s.cipher == nil {
		return nil, fmt.Errorf("channel service is not initialized")
	}
	if existing, err := s.repo.GetByName(ctx, input.Name); err == nil && existing != nil {
		return nil, domain.ErrChannelAlreadyExists
	}

	settingsJSON, err := normalizeJSONObject(input.Settings, true)
	if err != nil {
		return nil, err
	}
	encryptedCredentials, err := s.encryptCredentials(ctx, input.Credentials)
	if err != nil {
		return nil, err
	}
	if encryptedCredentials != "" {
		if err := verifyChannelConnectionByKind(ctx, strings.TrimSpace(strings.ToLower(input.Kind)), input.Credentials); err != nil {
			return nil, err
		}
	}

	channel, err := domain.NewChannel(
		newChannelID(),
		strings.TrimSpace(input.Name),
		strings.TrimSpace(input.Kind),
		strings.TrimSpace(input.Status),
		encryptedCredentials,
		settingsJSON,
	)
	if err != nil {
		return nil, err
	}

	saved, err := s.repo.Save(ctx, channel)
	if err != nil {
		return nil, err
	}
	return toChannelView(saved), nil
}

func (s *channelService) Update(ctx context.Context, input appservices.UpdateChannelInput) (*appservices.ChannelView, error) {
	if s.repo == nil || s.cipher == nil {
		return nil, fmt.Errorf("channel service is not initialized")
	}

	channel, err := s.repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if existing, err := s.repo.GetByName(ctx, input.Name); err == nil && existing != nil && existing.ID != input.ID {
		return nil, domain.ErrChannelAlreadyExists
	}

	settingsJSON, err := normalizeJSONObject(input.Settings, false)
	if err != nil {
		return nil, err
	}

	channel.Name = strings.TrimSpace(input.Name)
	channel.Kind = domain.ChannelKind(strings.TrimSpace(strings.ToLower(input.Kind)))
	channel.Status = domain.ChannelStatus(strings.TrimSpace(strings.ToLower(input.Status)))
	if settingsJSON != "" {
		channel.SettingsJSON = settingsJSON
	}
	if input.ReplaceCredentials {
		encryptedCredentials, err := s.encryptCredentials(ctx, input.Credentials)
		if err != nil {
			return nil, err
		}
		channel.EncryptedCredentials = encryptedCredentials
	}

	if input.ReplaceCredentials && channel.EncryptedCredentials != "" {
		decrypted, err := s.cipher.Decrypt(ctx, channel.EncryptedCredentials)
		if err != nil {
			return nil, domain.ErrChannelEncryptionFailed
		}
		if err := verifyChannelConnectionByKind(ctx, string(channel.Kind), json.RawMessage(decrypted)); err != nil {
			return nil, err
		}
	}
	channel.UpdatedAt = time.Now().UTC()

	if err := channel.Validate(); err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, channel)
	if err != nil {
		return nil, err
	}
	return toChannelView(updated), nil
}

func (s *channelService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *channelService) GetByID(ctx context.Context, id string) (*appservices.ChannelView, error) {
	channel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toChannelView(channel), nil
}

func (s *channelService) List(ctx context.Context, req common.BaseRequestModel) (*appservices.ChannelListView, error) {
	result, err := s.repo.GetAll(ctx, domain.ChannelListOptions{
		PageIndex:  req.PageIndex,
		PageSize:   req.PageSize,
		SearchText: req.SearchText,
		OrderBy:    req.OrderBy,
		Ascending:  req.Ascending,
	})
	if err != nil {
		return nil, err
	}

	items := make([]appservices.ChannelView, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, *toChannelView(item))
	}

	totalPage := 0
	if req.PageSize > 0 {
		totalPage = int((result.TotalItems + int64(req.PageSize) - 1) / int64(req.PageSize))
	}

	return &appservices.ChannelListView{
		Items:      items,
		PageIndex:  req.PageIndex,
		PageSize:   req.PageSize,
		TotalItems: result.TotalItems,
		TotalPage:  totalPage,
	}, nil
}

func (s *channelService) encryptCredentials(ctx context.Context, raw json.RawMessage) (string, error) {
	normalized, err := normalizeJSONObject(raw, true)
	if err != nil {
		return "", err
	}
	if normalized == "{}" {
		return "", nil
	}
	encrypted, err := s.cipher.Encrypt(ctx, normalized)
	if err != nil {
		return "", domain.ErrChannelEncryptionFailed
	}
	return encrypted, nil
}

func toChannelView(channel *domain.Channel) *appservices.ChannelView {
	if channel == nil {
		return nil
	}

	settings := json.RawMessage([]byte(channel.SettingsJSON))
	return &appservices.ChannelView{
		ID:             channel.ID,
		Name:           channel.Name,
		Kind:           string(channel.Kind),
		Status:         string(channel.Status),
		HasCredentials: strings.TrimSpace(channel.EncryptedCredentials) != "",
		Settings:       settings,
		CreatedAt:      channel.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      channel.UpdatedAt.Format(time.RFC3339),
	}
}

func normalizeJSONObject(raw json.RawMessage, allowDefault bool) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		if allowDefault {
			return "{}", nil
		}
		return "", nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", domain.ErrInvalidInput
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		return "", domain.ErrInvalidInput
	}
	return string(normalized), nil
}

func newChannelID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("channel-%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func verifyChannelConnectionByKind(ctx context.Context, kind string, raw json.RawMessage) error {
	switch domain.ChannelKind(strings.TrimSpace(strings.ToLower(kind))) {
	case domain.ChannelKindTelegram:
		_, err := testTelegramConnection(ctx, raw)
		return err
	case domain.ChannelKindDiscord:
		_, err := testDiscordConnection(ctx, raw)
		return err
	case domain.ChannelKindSlack:
		_, err := testSlackConnection(ctx, raw)
		return err
	case domain.ChannelKindWhatsApp, domain.ChannelKindWeb:
		return nil
	default:
		return domain.ErrUnsupportedChannelKind
	}
}
