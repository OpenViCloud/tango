package domain

import (
	"errors"
	"strings"
	"time"
)

type ChannelKind string
type ChannelStatus string

const (
	ChannelKindDiscord  ChannelKind = "discord"
	ChannelKindTelegram ChannelKind = "telegram"
	ChannelKindWhatsApp ChannelKind = "whatsapp"
	ChannelKindSlack    ChannelKind = "slack"
	ChannelKindWeb      ChannelKind = "web"

	ChannelStatusPending  ChannelStatus = "pending"
	ChannelStatusActive   ChannelStatus = "active"
	ChannelStatusDisabled ChannelStatus = "disabled"

	channelMaxIDLength                   = 64
	channelMaxNameLength                 = 100
	channelMaxKindLength                 = 32
	channelMaxStatusLength               = 32
	channelMaxEncryptedCredentialsLength = 8192
	channelMaxSettingsJSONLength         = 16384
)

var (
	ErrChannelNotFound         = errors.New("channel not found")
	ErrChannelAlreadyExists    = errors.New("channel already exists")
	ErrUnsupportedChannelKind  = errors.New("unsupported channel kind")
	ErrUnsupportedChannelState = errors.New("unsupported channel state")
	ErrChannelEncryptionFailed = errors.New("channel encryption failed")
)

type Channel struct {
	ID                   string
	Name                 string
	Kind                 ChannelKind
	Status               ChannelStatus
	EncryptedCredentials string
	SettingsJSON         string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            *time.Time
}

func NewChannel(id, name, kind, status, encryptedCredentials, settingsJSON string) (*Channel, error) {
	now := time.Now().UTC()
	channel := &Channel{
		ID:                   strings.TrimSpace(id),
		Name:                 strings.TrimSpace(name),
		Kind:                 ChannelKind(strings.TrimSpace(strings.ToLower(kind))),
		Status:               ChannelStatus(strings.TrimSpace(strings.ToLower(status))),
		EncryptedCredentials: strings.TrimSpace(encryptedCredentials),
		SettingsJSON:         strings.TrimSpace(settingsJSON),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := channel.Validate(); err != nil {
		return nil, err
	}
	return channel, nil
}

func (c *Channel) Validate() error {
	if c.ID == "" || c.Name == "" || c.SettingsJSON == "" {
		return ErrInvalidInput
	}
	if exceedsMaxLength(c.ID, channelMaxIDLength) ||
		exceedsMaxLength(c.Name, channelMaxNameLength) ||
		exceedsMaxLength(string(c.Kind), channelMaxKindLength) ||
		exceedsMaxLength(string(c.Status), channelMaxStatusLength) ||
		exceedsMaxLength(c.EncryptedCredentials, channelMaxEncryptedCredentialsLength) ||
		exceedsMaxLength(c.SettingsJSON, channelMaxSettingsJSONLength) {
		return ErrInvalidInput
	}

	switch c.Kind {
	case ChannelKindDiscord, ChannelKindTelegram, ChannelKindWhatsApp, ChannelKindSlack, ChannelKindWeb:
	default:
		return ErrUnsupportedChannelKind
	}

	switch c.Status {
	case ChannelStatusPending, ChannelStatusActive, ChannelStatusDisabled:
	default:
		return ErrUnsupportedChannelState
	}

	return nil
}

func exceedsMaxLength(value string, max int) bool {
	return len(value) > max
}
