package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	appservices "tango/internal/application/services"
	"tango/internal/messaging/inbound"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

const (
	telegramMessageLimit  = 4096
	telegramTypingAction  = 4 * time.Second
	callbackProjects      = "projects"
	callbackProjectPrefix = "project:"
	callbackEnvPrefix     = "env:"
	callbackResourcePrefx = "res:"
	callbackActionPrefix  = "act:"
	callbackBackProjects  = "back:projects"
	callbackBackProject   = "back:project:"
	callbackBackEnv       = "back:env:"
)

// Config contains runtime settings for the Telegram channel adapter.
type Config struct {
	ChannelID      string
	Token          string
	AllowedUserIDs map[string]bool
	EnableTyping   bool
	Navigator      appservices.TelegramProjectNavigator
}

// Channel adapts Telegram updates into normalized internal messages.
type Channel struct {
	cfg       Config
	bot       *telego.Bot
	publisher inbound.Publisher
	logger    *slog.Logger
	navigator appservices.TelegramProjectNavigator

	mu          sync.RWMutex
	rootCtx     context.Context
	cancel      context.CancelFunc
	started     bool
	botUserID   int64
	botUsername string
	wg          sync.WaitGroup
}

// New creates a new Telegram channel adapter.
func New(cfg Config, publisher inbound.Publisher, logger *slog.Logger) (*Channel, error) {
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, errors.New("telegram token is required")
	}
	if publisher == nil {
		return nil, errors.New("publisher is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	bot, err := telego.NewBot(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	return &Channel{
		cfg:       cfg,
		bot:       bot,
		publisher: publisher,
		logger:    logger,
		navigator: cfg.Navigator,
	}, nil
}

// Start begins polling Telegram updates.
func (c *Channel) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return errors.New("telegram channel already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	c.rootCtx = ctx
	c.cancel = cancel
	c.mu.Unlock()

	me, err := c.bot.GetMe(ctx)
	if err != nil {
		cancel()
		c.resetStartState()
		return fmt.Errorf("get telegram bot identity: %w", err)
	}

	updates, err := c.bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		cancel()
		c.resetStartState()
		return fmt.Errorf("start telegram long polling: %w", err)
	}

	c.mu.Lock()
	c.botUserID = me.ID
	c.botUsername = me.Username
	c.started = true
	c.mu.Unlock()

	c.logger.Info("telegram connected", "username", me.Username, "user_id", me.ID)
	c.logger.Info("telegram config", "enable_typing", c.cfg.EnableTyping, "allowed_users", len(c.cfg.AllowedUserIDs))

	c.wg.Add(1)
	go c.consumeUpdates(updates)
	return nil
}

// Stop stops polling Telegram updates and waits for the background loop to exit.
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

// Name returns the channel name.
func (c *Channel) Name() string {
	return "telegram"
}

func (c *Channel) consumeUpdates(updates <-chan telego.Update) {
	defer c.wg.Done()

	for update := range updates {
		switch {
		case update.Message != nil:
			c.handleMessage(update.Message)
		case update.CallbackQuery != nil:
			c.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (c *Channel) handleMessage(m *telego.Message) {
	if m == nil || m.From == nil {
		return
	}
	if m.From.IsBot {
		return
	}
	if isServiceMessage(m) {
		return
	}
	if !c.isAllowedUser(m.From.ID) {
		c.logger.Warn("telegram user not allowed", "user_id", m.From.ID)
		return
	}

	content := strings.TrimSpace(m.Text)
	if content == "" {
		content = strings.TrimSpace(m.Caption)
	}
	media := normalizeAttachments(m)

	c.logger.Info("telegram message received", "chat_id", m.Chat.ID, "author_id", m.From.ID, "content", content, "attachments", len(media))

	if strings.HasPrefix(strings.TrimSpace(content), "/") && c.handleLocalCommand(m, content) {
		return
	}

	inboundMsg := &inbound.Message{
		Channel:   c.Name(),
		ChannelID: c.cfg.ChannelID,
		SenderID:  strconv.FormatInt(m.From.ID, 10),
		Sender:    buildDisplayName(m.From),
		ChatID:    strconv.FormatInt(m.Chat.ID, 10),
		MessageID: strconv.Itoa(m.MessageID),
		Content:   content,
		Media:     media,
		Metadata: map[string]any{
			"chat_type":     m.Chat.Type,
			"chat_title":    m.Chat.Title,
			"chat_username": m.Chat.Username,
			"has_photo":     len(m.Photo) > 0,
			"has_document":  m.Document != nil,
		},
		Timestamp: time.Unix(m.Date, 0).UTC(),
	}

	ctx, cancel := context.WithTimeout(c.runCtx(), 30*time.Second)
	defer cancel()
	stopTyping := c.startTyping(ctx, m.Chat.ID)
	defer stopTyping()

	if err := c.publisher.PublishInbound(ctx, inboundMsg); err != nil {
		c.logger.Error("telegram publish inbound failed", "err", err)
		_, _ = c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: tu.ID(m.Chat.ID),
			Text:   "Sorry, I couldn't process that message.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: m.MessageID,
			},
		})
	}
}

// Send posts a final non-streaming message back to Telegram.
func (c *Channel) Send(ctx context.Context, msg *inbound.OutboundMessage) error {
	if msg == nil {
		return errors.New("outbound message is nil")
	}
	if strings.TrimSpace(msg.ChatID) == "" {
		return errors.New("outbound message ChatID is required")
	}

	chatID, err := strconv.ParseInt(strings.TrimSpace(msg.ChatID), 10, 64)
	if err != nil {
		return fmt.Errorf("parse telegram chat id: %w", err)
	}
	replyTo, err := parseReplyID(msg.ReplyTo)
	if err != nil {
		return err
	}

	parts := splitMessage(msg.Content, telegramMessageLimit)
	if len(parts) == 0 && len(msg.Files) > 0 {
		return errors.New("telegram outbound files are not implemented")
	}
	if len(parts) == 0 {
		return errors.New("outbound message has no content and no files")
	}

	for i, part := range parts {
		params := &telego.SendMessageParams{
			ChatID: tu.ID(chatID),
			Text:   part,
		}
		if i == 0 && replyTo != 0 {
			params.ReplyParameters = &telego.ReplyParameters{
				MessageID: replyTo,
			}
		}

		if _, err := c.bot.SendMessage(ctx, params); err != nil {
			return fmt.Errorf("send telegram message (chunk %d/%d): %w", i+1, len(parts), err)
		}
	}

	return nil
}

func (c *Channel) startTyping(ctx context.Context, chatID int64) func() {
	if !c.cfg.EnableTyping {
		return func() {}
	}

	typingCtx, cancel := context.WithCancel(ctx)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		ticker := time.NewTicker(telegramTypingAction)
		defer ticker.Stop()

		sendTyping := func() {
			if err := c.bot.SendChatAction(typingCtx, tu.ChatAction(tu.ID(chatID), telego.ChatActionTyping)); err != nil && !errors.Is(err, context.Canceled) {
				c.logger.Warn("telegram send typing failed", "err", err)
			}
		}

		sendTyping()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				sendTyping()
			}
		}
	}()

	return cancel
}

func (c *Channel) runCtx() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.rootCtx != nil {
		return c.rootCtx
	}
	return context.Background()
}

func (c *Channel) resetStartState() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rootCtx = nil
	c.cancel = nil
	c.started = false
}

func (c *Channel) isAllowedUser(userID int64) bool {
	if len(c.cfg.AllowedUserIDs) == 0 {
		return true
	}
	return c.cfg.AllowedUserIDs[strconv.FormatInt(userID, 10)]
}

func (c *Channel) handleLocalCommand(m *telego.Message, content string) bool {
	cmd := strings.ToLower(strings.Fields(strings.TrimSpace(content))[0])

	switch cmd {
	case "/help":
		return c.sendCommandReply(m, "Available local commands: /help, /status, /ping, /projects")
	case "/status":
		return c.sendCommandReply(m, "Telegram channel is running normally.")
	case "/ping":
		return c.sendCommandReply(m, "Pong.")
	case "/projects":
		return c.showProjectsMessage(m.Chat.ID, m.MessageID)
	default:
		return false
	}
}

func (c *Channel) sendCommandReply(m *telego.Message, reply string) bool {
	ctx, cancel := context.WithTimeout(c.runCtx(), 15*time.Second)
	defer cancel()

	_, _ = c.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: tu.ID(m.Chat.ID),
		Text:   reply,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: m.MessageID,
		},
	})
	return true
}

func (c *Channel) handleCallbackQuery(q *telego.CallbackQuery) {
	if q == nil || q.Message == nil {
		return
	}
	if !c.isAllowedUser(q.From.ID) {
		c.answerCallback(q.ID, "You are not allowed to use this bot.", true)
		c.logger.Warn("telegram callback user not allowed", "user_id", q.From.ID)
		return
	}

	chatID := q.Message.GetChat().ID
	messageID := q.Message.GetMessageID()
	data := strings.TrimSpace(q.Data)
	switch {
	case data == callbackProjects || data == callbackBackProjects:
		if err := c.editProjectsMessage(chatID, messageID); err != nil {
			c.logger.Error("telegram edit projects failed", "err", err)
			c.answerCallback(q.ID, "Couldn't load projects.", true)
			return
		}
		c.answerCallback(q.ID, "", false)
	case strings.HasPrefix(data, callbackProjectPrefix):
		projectID := strings.TrimPrefix(data, callbackProjectPrefix)
		if err := c.editProjectEnvironments(chatID, messageID, projectID); err != nil {
			c.logger.Error("telegram edit project environments failed", "err", err, "project_id", projectID)
			c.answerCallback(q.ID, "Couldn't load environments.", true)
			return
		}
		c.answerCallback(q.ID, "", false)
	case strings.HasPrefix(data, callbackEnvPrefix):
		envID := strings.TrimPrefix(data, callbackEnvPrefix)
		if strings.TrimSpace(envID) == "" {
			c.answerCallback(q.ID, "Invalid selection.", true)
			return
		}
		if err := c.editEnvironmentResources(chatID, messageID, envID); err != nil {
			c.logger.Error("telegram edit environment resources failed", "err", err, "environment_id", envID)
			c.answerCallback(q.ID, "Couldn't load resources.", true)
			return
		}
		c.answerCallback(q.ID, "", false)
	case strings.HasPrefix(data, callbackResourcePrefx):
		resourceID := strings.TrimPrefix(data, callbackResourcePrefx)
		if strings.TrimSpace(resourceID) == "" {
			c.answerCallback(q.ID, "Invalid selection.", true)
			return
		}
		if err := c.editResourceDetail(chatID, messageID, resourceID, ""); err != nil {
			c.logger.Error("telegram edit resource detail failed", "err", err, "resource_id", resourceID)
			c.answerCallback(q.ID, "Couldn't load resource.", true)
			return
		}
		c.answerCallback(q.ID, "", false)
	case strings.HasPrefix(data, callbackBackProject):
		projectID := strings.TrimPrefix(data, callbackBackProject)
		if err := c.editProjectEnvironments(chatID, messageID, projectID); err != nil {
			c.logger.Error("telegram back to project failed", "err", err, "project_id", projectID)
			c.answerCallback(q.ID, "Couldn't load environments.", true)
			return
		}
		c.answerCallback(q.ID, "", false)
	case strings.HasPrefix(data, callbackBackEnv):
		envID := strings.TrimPrefix(data, callbackBackEnv)
		if err := c.editEnvironmentResources(chatID, messageID, envID); err != nil {
			c.logger.Error("telegram back to environment failed", "err", err, "environment_id", envID)
			c.answerCallback(q.ID, "Couldn't load resources.", true)
			return
		}
		c.answerCallback(q.ID, "", false)
	case strings.HasPrefix(data, callbackActionPrefix):
		if err := c.handleResourceAction(chatID, messageID, q.ID, strings.TrimPrefix(data, callbackActionPrefix)); err != nil {
			c.logger.Error("telegram resource action failed", "err", err, "data", data)
			c.answerCallback(q.ID, "Action failed.", true)
			return
		}
	default:
		c.answerCallback(q.ID, "Unknown action.", false)
	}
}

func (c *Channel) answerCallback(queryID, text string, alert bool) {
	if strings.TrimSpace(queryID) == "" {
		return
	}

	ctx, cancel := context.WithTimeout(c.runCtx(), 5*time.Second)
	defer cancel()

	params := tu.CallbackQuery(queryID)
	if strings.TrimSpace(text) != "" {
		params = params.WithText(text)
	}
	if alert {
		params = params.WithShowAlert()
	}
	_ = c.bot.AnswerCallbackQuery(ctx, params)
}

func (c *Channel) showProjectsMessage(chatID int64, replyTo int) bool {
	ctx, cancel := context.WithTimeout(c.runCtx(), 15*time.Second)
	defer cancel()

	text, markup, err := c.renderProjectsView(ctx)
	if err != nil {
		c.logger.Error("telegram render projects failed", "err", err)
		_, _ = c.bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID: tu.ID(chatID),
			Text:   "Couldn't load projects right now.",
			ReplyParameters: &telego.ReplyParameters{
				MessageID: replyTo,
			},
		})
		return true
	}

	params := tu.Message(tu.ID(chatID), text).WithReplyMarkup(markup)
	if replyTo != 0 {
		params.ReplyParameters = &telego.ReplyParameters{MessageID: replyTo}
	}
	_, _ = c.bot.SendMessage(ctx, params)
	return true
}

func (c *Channel) editProjectsMessage(chatID int64, messageID int) error {
	ctx, cancel := context.WithTimeout(c.runCtx(), 15*time.Second)
	defer cancel()

	text, markup, err := c.renderProjectsView(ctx)
	if err != nil {
		return err
	}

	_, err = c.bot.EditMessageText(ctx, tu.EditMessageText(tu.ID(chatID), messageID, text).WithReplyMarkup(markup))
	return err
}

func (c *Channel) editProjectEnvironments(chatID int64, messageID int, projectID string) error {
	ctx, cancel := context.WithTimeout(c.runCtx(), 15*time.Second)
	defer cancel()

	project, err := c.getProject(ctx, projectID)
	if err != nil {
		return err
	}

	text, markup := renderProjectEnvironmentsView(project)
	_, err = c.bot.EditMessageText(ctx, tu.EditMessageText(tu.ID(chatID), messageID, text).WithReplyMarkup(markup))
	return err
}

func (c *Channel) editEnvironmentResources(chatID int64, messageID int, envID string) error {
	ctx, cancel := context.WithTimeout(c.runCtx(), 15*time.Second)
	defer cancel()

	project, selected, err := c.getProjectAndEnvironment(ctx, envID)
	if err != nil {
		return err
	}

	resources, err := c.listEnvironmentResources(ctx, envID)
	if err != nil {
		return err
	}

	text, markup := renderEnvironmentResourcesView(project, selected, resources)
	_, err = c.bot.EditMessageText(ctx, tu.EditMessageText(tu.ID(chatID), messageID, text).WithReplyMarkup(markup))
	return err
}

func (c *Channel) editResourceDetail(chatID int64, messageID int, resourceID, note string) error {
	ctx, cancel := context.WithTimeout(c.runCtx(), 15*time.Second)
	defer cancel()

	resource, err := c.getResource(ctx, resourceID)
	if err != nil {
		return err
	}

	project, env, err := c.getProjectAndEnvironment(ctx, resource.EnvironmentID)
	if err != nil {
		return err
	}

	text, markup := renderResourceDetailView(project, env, resource, note)
	_, err = c.bot.EditMessageText(ctx, tu.EditMessageText(tu.ID(chatID), messageID, text).WithReplyMarkup(markup))
	return err
}

func (c *Channel) handleResourceAction(chatID int64, messageID int, queryID, payload string) error {
	action, resourceID, ok := strings.Cut(payload, ":")
	if !ok || strings.TrimSpace(resourceID) == "" {
		return fmt.Errorf("invalid resource action payload %q", payload)
	}

	ctx, cancel := context.WithTimeout(c.runCtx(), 30*time.Second)
	defer cancel()

	var note string
	switch action {
	case "start":
		if err := c.navigator.StartResource(ctx, resourceID); err != nil {
			return err
		}
		note = "Start requested. Refresh in a moment if status has not changed yet."
	case "stop":
		if err := c.navigator.StopResource(ctx, resourceID); err != nil {
			return err
		}
		note = "Resource stopped."
	case "restart":
		if err := c.navigator.RestartResource(ctx, resourceID); err != nil {
			return err
		}
		note = "Restart requested. Refresh in a moment if status has not changed yet."
	case "refresh":
		note = ""
	default:
		return fmt.Errorf("unsupported action %q", action)
	}

	if err := c.editResourceDetail(chatID, messageID, resourceID, note); err != nil {
		return err
	}
	c.answerCallback(queryID, "", false)
	return nil
}

func (c *Channel) renderProjectsView(ctx context.Context) (string, *telego.InlineKeyboardMarkup, error) {
	projects, err := c.listProjects(ctx)
	if err != nil {
		return "", nil, err
	}
	if len(projects) == 0 {
		return "No projects available.", tu.InlineKeyboard(), nil
	}

	rows := make([][]telego.InlineKeyboardButton, 0, len(projects))
	for _, project := range projects {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(project.Name).WithCallbackData(callbackProjectPrefix+project.ID),
		))
	}

	return "Choose a project", tu.InlineKeyboard(rows...), nil
}

func renderProjectEnvironmentsView(project appservices.TelegramProject) (string, *telego.InlineKeyboardMarkup) {
	if len(project.Environments) == 0 {
		return fmt.Sprintf("Project: %s\nNo environments available.", project.Name), tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("Back").WithCallbackData(callbackBackProjects),
			),
		)
	}

	envs := slices.Clone(project.Environments)
	rows := make([][]telego.InlineKeyboardButton, 0, len(envs)+1)
	for _, env := range envs {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(env.Name).WithCallbackData(callbackEnvPrefix+env.ID),
		))
	}
	rows = append(rows, tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("Back").WithCallbackData(callbackBackProjects),
	))

	return fmt.Sprintf("Project: %s\nChoose an environment", project.Name), tu.InlineKeyboard(rows...)
}

func renderEnvironmentResourcesView(project appservices.TelegramProject, env appservices.TelegramProjectEnvironment, resources []appservices.TelegramResource) (string, *telego.InlineKeyboardMarkup) {
	rows := make([][]telego.InlineKeyboardButton, 0, len(resources)+1)
	for _, resource := range resources {
		label := resource.Name
		if status := strings.TrimSpace(resource.Status); status != "" {
			label = fmt.Sprintf("%s (%s)", resource.Name, status)
		}
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(label).WithCallbackData(callbackResourcePrefx+resource.ID),
		))
	}
	rows = append(rows, tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("Back").WithCallbackData(callbackBackProject+project.ID),
	))

	if len(resources) == 0 {
		return fmt.Sprintf("Project: %s\nEnvironment: %s\nNo resources available.", project.Name, env.Name), tu.InlineKeyboard(rows...)
	}
	return fmt.Sprintf("Project: %s\nEnvironment: %s\nChoose a resource", project.Name, env.Name), tu.InlineKeyboard(rows...)
}

func renderResourceDetailView(project appservices.TelegramProject, env appservices.TelegramProjectEnvironment, resource appservices.TelegramResource, note string) (string, *telego.InlineKeyboardMarkup) {
	var lines []string
	lines = append(lines,
		"Project: "+project.Name,
		"Environment: "+env.Name,
		"Resource: "+resource.Name,
		"Status: "+resource.Status,
		"Type: "+resource.Type,
	)
	if image := strings.TrimSpace(resource.Image); image != "" {
		imageRef := image
		if tag := strings.TrimSpace(resource.Tag); tag != "" {
			imageRef += ":" + tag
		}
		lines = append(lines, "Image: "+imageRef)
	}
	if containerID := strings.TrimSpace(resource.ContainerID); containerID != "" {
		lines = append(lines, "Container: "+containerID)
	}
	if len(resource.Ports) > 0 {
		ports := make([]string, 0, len(resource.Ports))
		for _, port := range resource.Ports {
			label := fmt.Sprintf("%d->%d/%s", port.HostPort, port.InternalPort, fallbackProto(port.Proto))
			if strings.TrimSpace(port.Label) != "" {
				label += " " + port.Label
			}
			ports = append(ports, label)
		}
		lines = append(lines, "Ports: "+strings.Join(ports, ", "))
	}
	if strings.TrimSpace(note) != "" {
		lines = append(lines, "", note)
	}

	rows := make([][]telego.InlineKeyboardButton, 0, 3)
	switch resource.Status {
	case "running":
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Stop").WithCallbackData(callbackActionPrefix+"stop:"+resource.ID),
			tu.InlineKeyboardButton("Restart").WithCallbackData(callbackActionPrefix+"restart:"+resource.ID),
		))
	default:
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Start").WithCallbackData(callbackActionPrefix+"start:"+resource.ID),
		))
	}
	rows = append(rows, tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("Refresh").WithCallbackData(callbackActionPrefix+"refresh:"+resource.ID),
	))
	rows = append(rows, tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("Back").WithCallbackData(callbackBackEnv+env.ID),
	))

	return strings.Join(lines, "\n"), tu.InlineKeyboard(rows...)
}

func (c *Channel) listProjects(ctx context.Context) ([]appservices.TelegramProject, error) {
	if c.navigator == nil {
		return nil, errors.New("telegram navigator is not configured")
	}
	return c.navigator.ListProjects(ctx)
}

func (c *Channel) listEnvironmentResources(ctx context.Context, environmentID string) ([]appservices.TelegramResource, error) {
	if c.navigator == nil {
		return nil, errors.New("telegram navigator is not configured")
	}
	return c.navigator.ListEnvironmentResources(ctx, environmentID)
}

func (c *Channel) getResource(ctx context.Context, resourceID string) (appservices.TelegramResource, error) {
	if c.navigator == nil {
		return appservices.TelegramResource{}, errors.New("telegram navigator is not configured")
	}
	return c.navigator.GetResource(ctx, resourceID)
}

func (c *Channel) getProject(ctx context.Context, projectID string) (appservices.TelegramProject, error) {
	projects, err := c.listProjects(ctx)
	if err != nil {
		return appservices.TelegramProject{}, err
	}
	for _, project := range projects {
		if project.ID == projectID {
			return project, nil
		}
	}
	return appservices.TelegramProject{}, fmt.Errorf("project %q not found", projectID)
}

func (c *Channel) getProjectAndEnvironment(ctx context.Context, envID string) (appservices.TelegramProject, appservices.TelegramProjectEnvironment, error) {
	projects, err := c.listProjects(ctx)
	if err != nil {
		return appservices.TelegramProject{}, appservices.TelegramProjectEnvironment{}, err
	}
	for _, project := range projects {
		for _, env := range project.Environments {
			if env.ID == envID {
				return project, env, nil
			}
		}
	}
	return appservices.TelegramProject{}, appservices.TelegramProjectEnvironment{}, fmt.Errorf("environment %q not found", envID)
}

func fallbackProto(proto string) string {
	if strings.TrimSpace(proto) == "" {
		return "tcp"
	}
	return proto
}

func isServiceMessage(m *telego.Message) bool {
	if m == nil {
		return false
	}
	return len(m.NewChatMembers) > 0 ||
		m.LeftChatMember != nil ||
		m.NewChatTitle != "" ||
		len(m.NewChatPhoto) > 0 ||
		m.DeleteChatPhoto ||
		m.GroupChatCreated
}

func normalizeAttachments(m *telego.Message) []inbound.Media {
	if m == nil {
		return nil
	}

	var out []inbound.Media
	if len(m.Photo) > 0 {
		best := m.Photo[len(m.Photo)-1]
		out = append(out, inbound.Media{
			Type:     inbound.MediaTypeImage,
			URL:      best.FileID,
			Filename: "photo",
			Size:     int(best.FileSize),
		})
	}
	if m.Document != nil {
		out = append(out, inbound.Media{
			Type:        inbound.MediaTypeDocument,
			URL:         m.Document.FileID,
			Filename:    m.Document.FileName,
			ContentType: m.Document.MimeType,
			Size:        int(m.Document.FileSize),
		})
	}
	if m.Video != nil {
		out = append(out, inbound.Media{
			Type:        inbound.MediaTypeVideo,
			URL:         m.Video.FileID,
			Filename:    m.Video.FileName,
			ContentType: m.Video.MimeType,
			Size:        int(m.Video.FileSize),
		})
	}
	if m.Audio != nil {
		out = append(out, inbound.Media{
			Type:        inbound.MediaTypeAudio,
			URL:         m.Audio.FileID,
			Filename:    m.Audio.FileName,
			ContentType: m.Audio.MimeType,
			Size:        int(m.Audio.FileSize),
		})
	}
	if m.Voice != nil {
		out = append(out, inbound.Media{
			Type:     inbound.MediaTypeAudio,
			URL:      m.Voice.FileID,
			Filename: "voice.ogg",
			Size:     int(m.Voice.FileSize),
		})
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func buildDisplayName(u *telego.User) string {
	if u == nil {
		return ""
	}
	fullName := strings.TrimSpace(strings.TrimSpace(u.FirstName + " " + u.LastName))
	if fullName != "" {
		return fullName
	}
	if strings.TrimSpace(u.Username) != "" {
		return u.Username
	}
	return strconv.FormatInt(u.ID, 10)
}

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

func parseReplyID(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	replyTo, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("parse telegram reply id: %w", err)
	}
	return replyTo, nil
}
