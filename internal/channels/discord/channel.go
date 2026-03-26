package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"tango/internal/messaging/inbound"

	"github.com/bwmarrin/discordgo"
)

const (
	// DiscordMessageLimit is the maximum number of characters allowed in one Discord message.
	DiscordMessageLimit = 2000

	// DefaultThinkingMessage is used as a placeholder while the bot is processing a request.
	DefaultThinkingMessage = "Thinking..."

	// HealthCheckInterval defines how often we verify that the Discord session is still alive.
	HealthCheckInterval = 30 * time.Second

	// streamEditThrottle is the minimum time between stream edits to reduce API pressure.
	streamEditThrottle = 700 * time.Millisecond

	// typingInterval is how often a typing indicator is re-sent while processing.
	typingInterval = 8 * time.Second
)

// Config contains runtime settings for the Discord channel adapter.
type Config struct {
	ChannelID string
	Token string

	// AllowedUserIDs restricts which users can talk to the bot.
	// Leave empty to allow everyone.
	AllowedUserIDs map[string]bool

	// RequireMention forces the bot to only respond when mentioned in guild channels.
	// Direct messages are always accepted.
	RequireMention bool

	// ThinkingMessage is sent before the final answer is ready.
	ThinkingMessage string

	// EnableTyping enables periodic "bot is typing..." feedback while processing.
	EnableTyping bool

	// EnableMessageContentIntent requests the privileged MESSAGE_CONTENT intent.
	// Enable it only when the bot must read non-mention messages in guild channels.
	EnableMessageContentIntent bool
}

// StreamChunk is used for streaming responses.
type StreamChunk struct {
	Content string
	Done    bool
	Err     error
}

// DiscordChannel adapts Discord events into a normalized internal message format.
type DiscordChannel struct {
	cfg       Config
	session   *discordgo.Session
	publisher inbound.Publisher
	logger    *slog.Logger

	mu           sync.RWMutex
	botUserID    string
	rootCtx      context.Context
	cancel       context.CancelFunc
	started      bool
	handlerAdded bool
	readyAdded   bool
}

// New creates a new Discord channel adapter.
func New(cfg Config, publisher inbound.Publisher, logger *slog.Logger) (*DiscordChannel, error) {
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, errors.New("discord token is required")
	}
	if publisher == nil {
		return nil, errors.New("publisher is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.ThinkingMessage == "" {
		cfg.ThinkingMessage = DefaultThinkingMessage
	}

	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}

	s.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages
	if cfg.EnableMessageContentIntent {
		s.Identify.Intents |= discordgo.IntentsMessageContent
	}

	return &DiscordChannel{
		cfg:       cfg,
		session:   s,
		publisher: publisher,
		logger:    logger,
	}, nil
}

// Start opens the Discord gateway connection and begins listening for events.
func (c *DiscordChannel) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return errors.New("discord channel already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	c.rootCtx = ctx
	c.cancel = cancel
	if !c.handlerAdded {
		c.session.AddHandler(c.handleMessageCreate)
		c.handlerAdded = true
	}
	if !c.readyAdded {
		c.session.AddHandler(c.handleReady)
		c.readyAdded = true
	}
	c.mu.Unlock()

	if err := c.session.Open(); err != nil {
		cancel()
		c.mu.Lock()
		c.rootCtx = nil
		c.cancel = nil
		c.mu.Unlock()
		return fmt.Errorf("open discord session: %w", err)
	}

	user, err := c.session.User("@me")
	if err != nil {
		cancel()
		_ = c.session.Close()
		c.mu.Lock()
		c.rootCtx = nil
		c.cancel = nil
		c.mu.Unlock()
		return fmt.Errorf("get bot identity: %w", err)
	}

	c.mu.Lock()
	c.botUserID = user.ID
	c.started = true
	c.mu.Unlock()

	c.logger.Info("discord connected", "username", user.Username, "user_id", user.ID)
	c.logger.Info("discord config",
		"require_mention", c.cfg.RequireMention,
		"enable_typing", c.cfg.EnableTyping,
		"enable_message_content_intent", c.cfg.EnableMessageContentIntent,
		"allowed_users", len(c.cfg.AllowedUserIDs),
	)

	go c.healthCheckLoop(ctx)
	return nil
}

// Stop closes the Discord session and stops background tasks.
func (c *DiscordChannel) Stop() error {
	c.mu.Lock()
	cancel := c.cancel
	c.cancel = nil
	c.rootCtx = nil
	c.started = false
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}

// Name returns the channel name.
func (c *DiscordChannel) Name() string {
	return "discord"
}

func (c *DiscordChannel) handleReady(_ *discordgo.Session, r *discordgo.Ready) {
	if r == nil || r.User == nil {
		return
	}
	c.logger.Info("discord ready",
		"username", r.User.Username,
		"user_id", r.User.ID,
		"guilds", len(r.Guilds),
		"private_channels", len(r.PrivateChannels),
	)
}

// handleMessageCreate converts a Discord message into an internal normalized message.
func (c *DiscordChannel) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m == nil || m.Author == nil {
		c.logger.Warn("discord skip message", "reason", "nil event or author")
		return
	}
	if m.Author.Bot {
		c.logger.Info("discord skip bot message", "author_id", m.Author.ID)
		return
	}
	if !c.isAllowedUser(m.Author.ID) {
		c.logger.Warn("discord user not allowed", "author_id", m.Author.ID)
		return
	}
	if !c.shouldProcessMessage(m.Message) {
		c.logger.Info("discord skip message", "reason", "mention required", "channel_id", m.ChannelID, "guild_id", m.GuildID, "author_id", m.Author.ID, "content", m.Content)
		return
	}

	c.logger.Info("discord message received",
		"channel_id", m.ChannelID,
		"guild_id", m.GuildID,
		"author_id", m.Author.ID,
		"content", m.Content,
		"attachments", len(m.Attachments),
	)

	cleanedContent := c.cleanContent(m.Message)
	if strings.HasPrefix(strings.TrimSpace(cleanedContent), "/") {
		if c.handleLocalCommand(m, cleanedContent) {
			return
		}
	}

	inboundMsg := &inbound.Message{
		Channel:   c.Name(),
		ChannelID: c.cfg.ChannelID,
		SenderID:  m.Author.ID,
		Sender:    buildDisplayName(m.Author),
		ChatID:    m.ChannelID,
		GuildID:   m.GuildID,
		MessageID: m.ID,
		Content:   cleanedContent,
		Media:     normalizeAttachments(m.Attachments),
		Metadata: map[string]any{
			"author_username":   m.Author.Username,
			"author_globalname": m.Author.GlobalName,
			"mention_everyone":  m.MentionEveryone,
			"mentions_count":    len(m.Mentions),
		},
		Timestamp: m.Timestamp,
	}

	ctx, cancel := context.WithTimeout(c.runCtx(), 30*time.Second)
	defer cancel()

	if err := c.publisher.PublishInbound(ctx, inboundMsg); err != nil {
		c.logger.Error("discord publish inbound failed", "err", err)
		_, _ = s.ChannelMessageSendReply(
			m.ChannelID,
			"Sorry, I couldn't process that message.",
			&discordgo.MessageReference{
				MessageID: m.ID,
				ChannelID: m.ChannelID,
				GuildID:   m.GuildID,
			},
		)
	}
}

// runCtx returns a context that is cancelled when the channel stops.
func (c *DiscordChannel) runCtx() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.rootCtx != nil {
		return c.rootCtx
	}
	return context.Background()
}

// shouldProcessMessage decides whether the bot should handle the message.
func (c *DiscordChannel) shouldProcessMessage(m *discordgo.Message) bool {
	if m == nil {
		return false
	}
	if m.GuildID == "" {
		return true
	}
	if c.cfg.RequireMention {
		return c.isMentioned(m)
	}
	return true
}

// cleanContent removes the bot mention from content to improve downstream prompts.
func (c *DiscordChannel) cleanContent(m *discordgo.Message) string {
	if m == nil {
		return ""
	}

	content := strings.TrimSpace(m.Content)
	botID := c.getBotUserID()
	if botID == "" {
		return content
	}

	content = strings.ReplaceAll(content, "<@"+botID+">", "")
	content = strings.ReplaceAll(content, "<@!"+botID+">", "")
	return strings.TrimSpace(content)
}

// handleLocalCommand handles local slash-like text commands such as /help and /status.
func (c *DiscordChannel) handleLocalCommand(m *discordgo.MessageCreate, cleanedContent string) bool {
	cmd := strings.ToLower(strings.TrimSpace(cleanedContent))
	ref := &discordgo.MessageReference{
		MessageID: m.ID,
		ChannelID: m.ChannelID,
		GuildID:   m.GuildID,
	}

	switch cmd {
	case "/help":
		_, _ = c.session.ChannelMessageSendReply(
			m.ChannelID,
			"Available local commands: /help, /status, /ping",
			ref,
		)
		return true
	case "/status":
		_, _ = c.session.ChannelMessageSendReply(
			m.ChannelID,
			"Discord channel is running normally.",
			ref,
		)
		return true
	case "/ping":
		_, _ = c.session.ChannelMessageSendReply(
			m.ChannelID,
			"Pong.",
			ref,
		)
		return true
	default:
		return false
	}
}

// Send posts a final non-streaming message back to Discord.
// It supports replies, file uploads, and automatic message chunking.
func (c *DiscordChannel) Send(ctx context.Context, msg *inbound.OutboundMessage) error {
	_ = ctx

	if msg == nil {
		return errors.New("outbound message is nil")
	}
	if strings.TrimSpace(msg.ChatID) == "" {
		return errors.New("outbound message ChatID is required")
	}

	if msg.Typing && c.cfg.EnableTyping {
		_ = c.session.ChannelTyping(msg.ChatID)
	}

	parts := splitMessage(msg.Content, DiscordMessageLimit)

	if len(parts) == 0 && len(msg.Files) > 0 {
		payload := &discordgo.MessageSend{}
		if msg.ReplyTo != "" {
			payload.Reference = &discordgo.MessageReference{
				MessageID: msg.ReplyTo,
				ChannelID: msg.ChatID,
			}
		}
		for _, f := range msg.Files {
			if f.Reader == nil {
				continue
			}
			payload.Files = append(payload.Files, &discordgo.File{
				Name:   f.Name,
				Reader: f.Reader,
			})
		}
		if len(payload.Files) == 0 {
			return errors.New("outbound message has no content and no readable files")
		}
		if _, err := c.session.ChannelMessageSendComplex(msg.ChatID, payload); err != nil {
			return fmt.Errorf("send discord files: %w", err)
		}
		return nil
	}

	for i, part := range parts {
		payload := &discordgo.MessageSend{
			Content: part,
		}

		if i == 0 {
			if msg.ReplyTo != "" {
				payload.Reference = &discordgo.MessageReference{
					MessageID: msg.ReplyTo,
					ChannelID: msg.ChatID,
				}
			}
			for _, f := range msg.Files {
				if f.Reader == nil {
					continue
				}
				payload.Files = append(payload.Files, &discordgo.File{
					Name:   f.Name,
					Reader: f.Reader,
				})
			}
		}

		if _, err := c.session.ChannelMessageSendComplex(msg.ChatID, payload); err != nil {
			return fmt.Errorf("send discord message (chunk %d/%d): %w", i+1, len(parts), err)
		}
	}

	if len(parts) == 0 {
		return errors.New("outbound message has no content and no files")
	}

	return nil
}

// SendStream sends a placeholder message and progressively edits it as chunks arrive.
func (c *DiscordChannel) SendStream(ctx context.Context, chatID string, replyTo string, stream <-chan StreamChunk) error {
	if strings.TrimSpace(chatID) == "" {
		return errors.New("chatID is required")
	}

	if c.cfg.EnableTyping {
		typingCtx, typingCancel := context.WithCancel(ctx)
		defer typingCancel()
		go c.typingLoop(typingCtx, chatID)
	}

	var ref *discordgo.MessageReference
	if replyTo != "" {
		ref = &discordgo.MessageReference{
			MessageID: replyTo,
			ChannelID: chatID,
		}
	}

	placeholder, err := c.session.ChannelMessageSendComplex(chatID, &discordgo.MessageSend{
		Content:   c.cfg.ThinkingMessage,
		Reference: ref,
	})
	if err != nil {
		return fmt.Errorf("send placeholder: %w", err)
	}

	var builder strings.Builder
	lastEdit := time.Now()

	editNow := func(force bool) error {
		if !force && time.Since(lastEdit) < streamEditThrottle {
			return nil
		}

		content := builder.String()
		if strings.TrimSpace(content) == "" {
			content = c.cfg.ThinkingMessage
		}

		firstChunk := content
		if runes := []rune(content); len(runes) > DiscordMessageLimit {
			firstChunk = string(runes[:DiscordMessageLimit])
		}

		_, err := c.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:      placeholder.ID,
			Channel: chatID,
			Content: &firstChunk,
		})
		if err != nil {
			return err
		}

		lastEdit = time.Now()
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			cancelledMsg := "Request canceled."
			_, _ = c.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:      placeholder.ID,
				Channel: chatID,
				Content: &cancelledMsg,
			})
			return ctx.Err()

		case chunk, ok := <-stream:
			if !ok {
				return finalizeStreamMessage(c.session, chatID, placeholder.ID, builder.String())
			}

			if chunk.Err != nil {
				errMsg := "An error occurred while generating the response."
				_, _ = c.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
					ID:      placeholder.ID,
					Channel: chatID,
					Content: &errMsg,
				})
				return chunk.Err
			}

			if chunk.Content != "" {
				builder.WriteString(chunk.Content)
				if err := editNow(false); err != nil {
					c.logger.Warn("discord stream edit failed", "err", err)
				}
			}

			if chunk.Done {
				return finalizeStreamMessage(c.session, chatID, placeholder.ID, builder.String())
			}
		}
	}
}

// finalizeStreamMessage writes the final content to the placeholder and sends any overflow as follow-up messages.
func finalizeStreamMessage(s *discordgo.Session, chatID, messageID, content string) error {
	if strings.TrimSpace(content) == "" {
		content = "Done."
	}

	chunks := splitMessage(content, DiscordMessageLimit)
	if len(chunks) == 0 {
		chunks = []string{"Done."}
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      messageID,
		Channel: chatID,
		Content: &chunks[0],
	}); err != nil {
		return fmt.Errorf("final stream edit: %w", err)
	}

	for i := 1; i < len(chunks); i++ {
		if _, err := s.ChannelMessageSend(chatID, chunks[i]); err != nil {
			return fmt.Errorf("send overflow chunk %d: %w", i, err)
		}
	}

	return nil
}

// AddReaction adds an emoji reaction to a message.
func (c *DiscordChannel) AddReaction(channelID, messageID, emoji string) error {
	return c.session.MessageReactionAdd(channelID, messageID, normalizeEmoji(emoji))
}

// RemoveReaction removes a specific user's reaction from a message.
func (c *DiscordChannel) RemoveReaction(channelID, messageID, emoji, userID string) error {
	return c.session.MessageReactionRemove(channelID, messageID, normalizeEmoji(emoji), userID)
}

// RemoveOwnReaction removes the bot's own reaction from a message.
func (c *DiscordChannel) RemoveOwnReaction(channelID, messageID, emoji string) error {
	return c.session.MessageReactionRemove(channelID, messageID, normalizeEmoji(emoji), "@me")
}

// GetReactionSummary returns a simple count summary of reactions on a message.
func (c *DiscordChannel) GetReactionSummary(channelID, messageID string) (map[string]int, error) {
	msg, err := c.session.ChannelMessage(channelID, messageID)
	if err != nil {
		return nil, err
	}

	summary := make(map[string]int, len(msg.Reactions))
	for _, r := range msg.Reactions {
		summary[formatEmoji(r.Emoji)] = r.Count
	}
	return summary, nil
}

// healthCheckLoop periodically verifies that the session still has a ready state.
func (c *DiscordChannel) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.session == nil || c.session.State == nil {
				c.logger.Warn("discord health check", "reason", "session state is nil")
				continue
			}

			c.session.State.RLock()
			ready := c.session.State.Ready.Version > 0
			c.session.State.RUnlock()

			if !ready {
				c.logger.Warn("discord health check", "reason", "session not ready")
			}
		}
	}
}

// typingLoop continuously sends typing indicators until the context is cancelled.
func (c *DiscordChannel) typingLoop(ctx context.Context, channelID string) {
	_ = c.session.ChannelTyping(channelID)

	ticker := time.NewTicker(typingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = c.session.ChannelTyping(channelID)
		}
	}
}

func (c *DiscordChannel) isAllowedUser(userID string) bool {
	if len(c.cfg.AllowedUserIDs) == 0 {
		return true
	}
	return c.cfg.AllowedUserIDs[userID]
}

func (c *DiscordChannel) isMentioned(m *discordgo.Message) bool {
	if m == nil {
		return false
	}

	botID := c.getBotUserID()
	if botID == "" {
		return false
	}

	for _, u := range m.Mentions {
		if u != nil && u.ID == botID {
			return true
		}
	}

	return strings.Contains(m.Content, "<@"+botID+">") ||
		strings.Contains(m.Content, "<@!"+botID+">")
}

func (c *DiscordChannel) getBotUserID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.botUserID
}

// normalizeAttachments converts Discord attachments into normalized media types.
func normalizeAttachments(atts []*discordgo.MessageAttachment) []inbound.Media {
	if len(atts) == 0 {
		return nil
	}

	out := make([]inbound.Media, 0, len(atts))
	for _, att := range atts {
		if att == nil {
			continue
		}

		ct := strings.ToLower(att.ContentType)
		mediaType := inbound.MediaTypeDocument

		switch {
		case strings.HasPrefix(ct, "image/"):
			mediaType = inbound.MediaTypeImage
		case strings.HasPrefix(ct, "video/"):
			mediaType = inbound.MediaTypeVideo
		case strings.HasPrefix(ct, "audio/"):
			mediaType = inbound.MediaTypeAudio
		}

		out = append(out, inbound.Media{
			Type:        mediaType,
			URL:         att.URL,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
		})
	}

	return out
}

// splitMessage breaks long text into Discord-safe chunks, preferring to split on newlines or whitespace.
func splitMessage(content string, limit int) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	runes := []rune(content)
	if len(runes) <= limit {
		return []string{content}
	}

	var parts []string
	start := 0

	for start < len(runes) {
		end := start + limit
		if end > len(runes) {
			end = len(runes)
		}

		if end < len(runes) {
			boundary := end
			for i := end; i > start+limit/2; i-- {
				if runes[i-1] == '\n' || runes[i-1] == ' ' {
					boundary = i
					break
				}
			}
			end = boundary
		}

		parts = append(parts, strings.TrimSpace(string(runes[start:end])))
		start = end
	}

	return parts
}

func buildDisplayName(u *discordgo.User) string {
	if u == nil {
		return ""
	}
	if strings.TrimSpace(u.GlobalName) != "" {
		return u.GlobalName
	}
	return u.Username
}

// normalizeEmoji converts common emoji input to the format expected by Discord APIs.
func normalizeEmoji(emoji string) string {
	return strings.TrimSpace(emoji)
}

// formatEmoji renders a reaction emoji into a stable, readable string.
func formatEmoji(e *discordgo.Emoji) string {
	if e == nil {
		return ""
	}
	if e.ID != "" {
		return fmt.Sprintf("%s:%s", e.Name, e.ID)
	}
	return e.Name
}
