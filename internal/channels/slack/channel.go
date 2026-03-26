package slack

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"tango/internal/messaging/inbound"

	goslack "github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type Config struct {
	ChannelID      string
	BotToken       string
	AppToken       string
	AllowedUserIDs map[string]bool
	RequireMention bool
	EnableTyping   bool
}

type Channel struct {
	cfg       Config
	api       *goslack.Client
	socket    *socketmode.Client
	publisher inbound.Publisher
	logger    *slog.Logger

	mu        sync.RWMutex
	rootCtx   context.Context
	cancel    context.CancelFunc
	started   bool
	wg        sync.WaitGroup
	botUserID string
	botName   string
}

func New(cfg Config, publisher inbound.Publisher, logger *slog.Logger) (*Channel, error) {
	if strings.TrimSpace(cfg.BotToken) == "" {
		return nil, errors.New("slack bot token is required")
	}
	if strings.TrimSpace(cfg.AppToken) == "" {
		return nil, errors.New("slack app token is required")
	}
	if !strings.HasPrefix(strings.TrimSpace(cfg.BotToken), "xoxb-") {
		return nil, errors.New("slack bot token must start with xoxb-")
	}
	if !strings.HasPrefix(strings.TrimSpace(cfg.AppToken), "xapp-") {
		return nil, errors.New("slack app token must start with xapp-")
	}
	if publisher == nil {
		return nil, errors.New("publisher is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	api := goslack.New(
		strings.TrimSpace(cfg.BotToken),
		goslack.OptionAppLevelToken(strings.TrimSpace(cfg.AppToken)),
		goslack.OptionRetry(3),
	)
	socket := socketmode.New(api)

	return &Channel{
		cfg:       cfg,
		api:       api,
		socket:    socket,
		publisher: publisher,
		logger:    logger,
	}, nil
}

func (c *Channel) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return errors.New("slack channel already started")
	}
	ctx, cancel := context.WithCancel(ctx)
	c.rootCtx = ctx
	c.cancel = cancel
	c.mu.Unlock()

	auth, err := c.api.AuthTestContext(ctx)
	if err != nil {
		cancel()
		c.resetStartState()
		return fmt.Errorf("slack auth test: %w", err)
	}

	c.mu.Lock()
	c.botUserID = auth.UserID
	c.botName = auth.User
	c.started = true
	c.mu.Unlock()

	c.logger.Info("slack connected",
		"user", auth.User,
		"user_id", auth.UserID,
		"team", auth.Team,
		"team_id", auth.TeamID,
	)
	c.logger.Info("slack config",
		"require_mention", c.cfg.RequireMention,
		"enable_typing", c.cfg.EnableTyping,
		"allowed_users", len(c.cfg.AllowedUserIDs),
	)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := c.socket.RunContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
			c.logger.Error("slack socket mode exited", "err", err)
		}
	}()

	c.wg.Add(1)
	go c.consumeEvents(ctx)
	return nil
}

func (c *Channel) Stop() error {
	c.mu.Lock()
	cancel := c.cancel
	c.cancel = nil
	c.rootCtx = nil
	c.started = false
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	c.wg.Wait()
	return nil
}

func (c *Channel) Name() string {
	return "slack"
}

func (c *Channel) Send(ctx context.Context, msg *inbound.OutboundMessage) error {
	if msg == nil {
		return errors.New("outbound message is nil")
	}
	if strings.TrimSpace(msg.ChatID) == "" {
		return errors.New("outbound message ChatID is required")
	}
	if strings.TrimSpace(msg.Content) == "" && len(msg.Files) == 0 {
		return errors.New("outbound message has no content and no files")
	}
	if len(msg.Files) > 0 {
		return errors.New("slack outbound files are not implemented")
	}

	opts := []goslack.MsgOption{
		goslack.MsgOptionText(msg.Content, false),
	}
	if threadTS := resolveThreadTS(msg); threadTS != "" {
		opts = append(opts, goslack.MsgOptionTS(threadTS))
	}

	if _, _, err := c.api.PostMessageContext(ctx, msg.ChatID, opts...); err != nil {
		return fmt.Errorf("send slack message: %w", err)
	}
	return nil
}

func (c *Channel) consumeEvents(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-c.socket.Events:
			if !ok {
				return
			}
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				c.logger.Info("slack socket connecting")
			case socketmode.EventTypeConnected:
				c.logger.Info("slack socket connected")
			case socketmode.EventTypeConnectionError:
				c.logger.Warn("slack socket connection error", "data", evt.Data)
			case socketmode.EventTypeEventsAPI:
				c.handleEventsAPI(ctx, evt)
			}
		}
	}
}

func (c *Channel) handleEventsAPI(ctx context.Context, evt socketmode.Event) {
	eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		c.logger.Debug("slack ignored unknown events api payload")
		return
	}
	if evt.Request != nil {
		c.socket.Ack(*evt.Request)
	}
	if eventsAPIEvent.Type != slackevents.CallbackEvent {
		return
	}

	switch ev := eventsAPIEvent.InnerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		c.handleAppMention(ctx, eventsAPIEvent, ev)
	case *slackevents.MessageEvent:
		c.handleMessageEvent(ctx, eventsAPIEvent, ev)
	}
}

func (c *Channel) handleAppMention(ctx context.Context, outer slackevents.EventsAPIEvent, ev *slackevents.AppMentionEvent) {
	if ev == nil {
		return
	}
	if !c.isAllowedUser(ev.User) {
		c.logger.Warn("slack user not allowed", "user_id", ev.User)
		return
	}

	content := strings.TrimSpace(c.cleanContent(ev.Text))
	if strings.HasPrefix(content, "/") && c.handleLocalCommand(ctx, ev, content) {
		return
	}

	threadTS := strings.TrimSpace(ev.ThreadTimeStamp)
	if threadTS == "" {
		threadTS = strings.TrimSpace(ev.TimeStamp)
	}

	c.logger.Info("slack message received",
		"team_id", outer.TeamID,
		"channel_id", ev.Channel,
		"user_id", ev.User,
		"thread_ts", threadTS,
		"content", content,
	)

	inboundMsg := &inbound.Message{
		Channel:   c.Name(),
		ChannelID: c.cfg.ChannelID,
		SenderID:  ev.User,
		Sender:    ev.User,
		ChatID:    ev.Channel,
		MessageID: strings.TrimSpace(ev.TimeStamp),
		Content:   content,
		Metadata: map[string]any{
			"team_id":   outer.TeamID,
			"thread_ts": threadTS,
		},
	}

	publishCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := c.publisher.PublishInbound(publishCtx, inboundMsg); err != nil {
		c.logger.Error("slack publish inbound failed", "err", err)
		_, _, _ = c.api.PostMessageContext(ctx, ev.Channel,
			goslack.MsgOptionText("Sorry, I couldn't process that message.", false),
			goslack.MsgOptionTS(threadTS),
		)
	}
}

func (c *Channel) handleMessageEvent(ctx context.Context, outer slackevents.EventsAPIEvent, ev *slackevents.MessageEvent) {
	if ev == nil {
		return
	}
	if !c.shouldProcessMessageEvent(ev) {
		return
	}
	if !c.isAllowedUser(ev.User) {
		c.logger.Warn("slack user not allowed", "user_id", ev.User)
		return
	}

	content := strings.TrimSpace(c.cleanContent(ev.Text))
	threadTS := strings.TrimSpace(ev.ThreadTimeStamp)
	if threadTS == "" {
		threadTS = strings.TrimSpace(ev.TimeStamp)
	}

	c.logger.Info("slack message received",
		"team_id", outer.TeamID,
		"channel_id", ev.Channel,
		"user_id", ev.User,
		"thread_ts", threadTS,
		"channel_type", ev.ChannelType,
		"content", content,
	)

	if strings.HasPrefix(content, "/") && c.handleMessageCommand(ctx, ev, content, threadTS) {
		return
	}

	inboundMsg := &inbound.Message{
		Channel:   c.Name(),
		ChannelID: c.cfg.ChannelID,
		SenderID:  ev.User,
		Sender:    ev.User,
		ChatID:    ev.Channel,
		MessageID: strings.TrimSpace(ev.TimeStamp),
		Content:   content,
		Metadata: map[string]any{
			"team_id":      outer.TeamID,
			"thread_ts":    threadTS,
			"channel_type": ev.ChannelType,
		},
	}

	publishCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := c.publisher.PublishInbound(publishCtx, inboundMsg); err != nil {
		c.logger.Error("slack publish inbound failed", "err", err)
		_, _, _ = c.api.PostMessageContext(ctx, ev.Channel,
			goslack.MsgOptionText("Sorry, I couldn't process that message.", false),
			goslack.MsgOptionTS(threadTS),
		)
	}
}

func (c *Channel) handleLocalCommand(ctx context.Context, ev *slackevents.AppMentionEvent, cleanedContent string) bool {
	if ev == nil {
		return false
	}
	threadTS := strings.TrimSpace(ev.ThreadTimeStamp)
	if threadTS == "" {
		threadTS = strings.TrimSpace(ev.TimeStamp)
	}

	var reply string
	switch strings.ToLower(strings.TrimSpace(cleanedContent)) {
	case "/help":
		reply = "Available local commands: /help, /status, /ping"
	case "/status":
		reply = "Slack channel is running normally."
	case "/ping":
		reply = "Pong."
	default:
		return false
	}

	_, _, _ = c.api.PostMessageContext(ctx, ev.Channel,
		goslack.MsgOptionText(reply, false),
		goslack.MsgOptionTS(threadTS),
	)
	return true
}

func (c *Channel) handleMessageCommand(ctx context.Context, ev *slackevents.MessageEvent, cleanedContent, threadTS string) bool {
	if ev == nil {
		return false
	}

	var reply string
	switch strings.ToLower(strings.TrimSpace(cleanedContent)) {
	case "/help":
		reply = "Available local commands: /help, /status, /ping"
	case "/status":
		reply = "Slack channel is running normally."
	case "/ping":
		reply = "Pong."
	default:
		return false
	}

	_, _, _ = c.api.PostMessageContext(ctx, ev.Channel,
		goslack.MsgOptionText(reply, false),
		goslack.MsgOptionTS(threadTS),
	)
	return true
}

func (c *Channel) cleanContent(content string) string {
	content = strings.TrimSpace(content)
	botUserID := c.getBotUserID()
	if botUserID == "" {
		return content
	}
	content = strings.ReplaceAll(content, "<@"+botUserID+">", "")
	return strings.TrimSpace(content)
}

func (c *Channel) shouldSkipMessageEvent(ev *slackevents.MessageEvent) bool {
	if ev == nil {
		return true
	}
	if strings.TrimSpace(ev.User) == "" {
		return true
	}
	if strings.TrimSpace(ev.BotID) != "" {
		return true
	}
	if strings.TrimSpace(ev.SubType) != "" {
		return true
	}
	if strings.TrimSpace(ev.Text) == "" {
		return true
	}
	return false
}

func (c *Channel) shouldProcessMessageEvent(ev *slackevents.MessageEvent) bool {
	if c.shouldSkipMessageEvent(ev) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(ev.ChannelType), "im") {
		return true
	}
	return c.isMentionedText(ev.Text)
}

func (c *Channel) isMentionedText(text string) bool {
	botUserID := c.getBotUserID()
	if botUserID == "" {
		return false
	}
	text = strings.TrimSpace(text)
	return strings.Contains(text, "<@"+botUserID+">")
}

func (c *Channel) getBotUserID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.botUserID
}

func (c *Channel) isAllowedUser(userID string) bool {
	if len(c.cfg.AllowedUserIDs) == 0 {
		return true
	}
	return c.cfg.AllowedUserIDs[strings.TrimSpace(userID)]
}

func (c *Channel) resetStartState() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rootCtx = nil
	c.cancel = nil
	c.started = false
}

func resolveThreadTS(msg *inbound.OutboundMessage) string {
	if msg == nil {
		return ""
	}
	if msg.Metadata != nil {
		if threadTS, ok := msg.Metadata["thread_ts"].(string); ok {
			return strings.TrimSpace(threadTS)
		}
	}
	return strings.TrimSpace(msg.ReplyTo)
}
