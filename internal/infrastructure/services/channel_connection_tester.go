package services

import (
	"context"
	"encoding/json"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"

	"github.com/bwmarrin/discordgo"
	goslack "github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"

	"github.com/mymmrac/telego"
)

const (
	channelConnectionInvalidPayloadCode       = "CHANNEL_TEST_CONNECTION_INVALID_PAYLOAD"
	channelConnectionTelegramTokenRequired    = "CHANNEL_TEST_CONNECTION_TELEGRAM_TOKEN_REQUIRED"
	channelConnectionTelegramAuthFailed       = "CHANNEL_TEST_CONNECTION_TELEGRAM_AUTH_FAILED"
	channelConnectionDiscordTokenRequired     = "CHANNEL_TEST_CONNECTION_DISCORD_TOKEN_REQUIRED"
	channelConnectionDiscordAuthFailed        = "CHANNEL_TEST_CONNECTION_DISCORD_AUTH_FAILED"
	channelConnectionSlackBotTokenRequired    = "CHANNEL_TEST_CONNECTION_SLACK_BOT_TOKEN_REQUIRED"
	channelConnectionSlackAppTokenRequired    = "CHANNEL_TEST_CONNECTION_SLACK_APP_TOKEN_REQUIRED"
	channelConnectionSlackAuthFailed          = "CHANNEL_TEST_CONNECTION_SLACK_AUTH_FAILED"
	channelConnectionSlackSocketModeFailed    = "CHANNEL_TEST_CONNECTION_SLACK_SOCKET_MODE_FAILED"
	channelConnectionUnsupportedKindCode      = "CHANNEL_TEST_CONNECTION_UNSUPPORTED_KIND"
)

func (s *channelService) TestConnection(ctx context.Context, input appservices.TestChannelConnectionInput) (*appservices.TestChannelConnectionView, error) {
	kind := domain.ChannelKind(strings.TrimSpace(strings.ToLower(input.Kind)))
	switch kind {
	case domain.ChannelKindTelegram:
		return testTelegramConnection(ctx, input.Credentials)
	case domain.ChannelKindDiscord:
		return testDiscordConnection(ctx, input.Credentials)
	case domain.ChannelKindSlack:
		return testSlackConnection(ctx, input.Credentials)
	case domain.ChannelKindWhatsApp, domain.ChannelKindWeb:
		return nil, newChannelConnectionError(channelConnectionUnsupportedKindCode, "Channel kind does not support connection testing.", domain.ErrUnsupportedChannelKind)
	default:
		return nil, newChannelConnectionError(channelConnectionUnsupportedKindCode, "Channel kind does not support connection testing.", domain.ErrUnsupportedChannelKind)
	}
}

func testTelegramConnection(ctx context.Context, raw json.RawMessage) (*appservices.TestChannelConnectionView, error) {
	var creds telegramChannelCredentials
	if err := decodeRawCredentials(raw, &creds); err != nil {
		return nil, err
	}
	if strings.TrimSpace(creds.Token) == "" {
		return nil, newChannelConnectionError(channelConnectionTelegramTokenRequired, "Telegram bot token is required.", domain.ErrInvalidInput)
	}
	bot, err := telego.NewBot(strings.TrimSpace(creds.Token))
	if err != nil {
		return nil, newChannelConnectionError(channelConnectionTelegramAuthFailed, "Telegram bot token is invalid or could not be verified.", err)
	}
	me, err := bot.GetMe(ctx)
	if err != nil {
		return nil, newChannelConnectionError(channelConnectionTelegramAuthFailed, "Telegram bot token is invalid or could not be verified.", err)
	}
	return &appservices.TestChannelConnectionView{
		Kind:    string(domain.ChannelKindTelegram),
		OK:      true,
		Message: "Telegram connection verified.",
		Details: map[string]any{
			"bot_id":   me.ID,
			"username": me.Username,
			"name":     strings.TrimSpace(strings.TrimSpace(me.FirstName + " " + me.LastName)),
		},
	}, nil
}

func testDiscordConnection(ctx context.Context, raw json.RawMessage) (*appservices.TestChannelConnectionView, error) {
	var creds discordChannelCredentials
	if err := decodeRawCredentials(raw, &creds); err != nil {
		return nil, err
	}
	if strings.TrimSpace(creds.Token) == "" {
		return nil, newChannelConnectionError(channelConnectionDiscordTokenRequired, "Discord bot token is required.", domain.ErrInvalidInput)
	}
	session, err := discordgo.New("Bot " + strings.TrimSpace(creds.Token))
	if err != nil {
		return nil, newChannelConnectionError(channelConnectionDiscordAuthFailed, "Discord bot token is invalid or could not be verified.", err)
	}
	user, err := session.User("@me", discordgo.WithContext(ctx))
	if err != nil {
		return nil, newChannelConnectionError(channelConnectionDiscordAuthFailed, "Discord bot token is invalid or could not be verified.", err)
	}
	return &appservices.TestChannelConnectionView{
		Kind:    string(domain.ChannelKindDiscord),
		OK:      true,
		Message: "Discord connection verified.",
		Details: map[string]any{
			"user_id":  user.ID,
			"username": user.Username,
		},
	}, nil
}

func testSlackConnection(ctx context.Context, raw json.RawMessage) (*appservices.TestChannelConnectionView, error) {
	var creds slackChannelCredentials
	if err := decodeRawCredentials(raw, &creds); err != nil {
		return nil, err
	}
	if strings.TrimSpace(creds.BotToken) == "" {
		return nil, newChannelConnectionError(channelConnectionSlackBotTokenRequired, "Slack bot token is required.", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(creds.AppToken) == "" {
		return nil, newChannelConnectionError(channelConnectionSlackAppTokenRequired, "Slack app token is required.", domain.ErrInvalidInput)
	}
	api := goslack.New(
		strings.TrimSpace(creds.BotToken),
		goslack.OptionAppLevelToken(strings.TrimSpace(creds.AppToken)),
		goslack.OptionRetry(1),
	)
	auth, err := api.AuthTestContext(ctx)
	if err != nil {
		return nil, newChannelConnectionError(channelConnectionSlackAuthFailed, "Slack bot token is invalid or could not be verified.", err)
	}
	socketClient := socketmode.New(api)
	info, _, err := socketClient.OpenContext(ctx)
	if err != nil {
		return nil, newChannelConnectionError(channelConnectionSlackSocketModeFailed, "Slack app token is invalid or Socket Mode could not be opened.", err)
	}
	return &appservices.TestChannelConnectionView{
		Kind:    string(domain.ChannelKindSlack),
		OK:      true,
		Message: "Slack connection verified.",
		Details: map[string]any{
			"user_id":       auth.UserID,
			"user":          auth.User,
			"team_id":       auth.TeamID,
			"team":          auth.Team,
			"websocket_url": info.URL,
		},
	}, nil
}

func decodeRawCredentials(raw json.RawMessage, out any) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return newChannelConnectionError(channelConnectionInvalidPayloadCode, "Credentials payload is required.", domain.ErrInvalidInput)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return newChannelConnectionError(channelConnectionInvalidPayloadCode, "Credentials payload must be a valid JSON object.", err)
	}
	return nil
}

func newChannelConnectionError(code, message string, cause error) error {
	return &appservices.ChannelConnectionError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
