package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tango/internal/messaging/inbound"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Config contains runtime settings for the WhatsApp channel adapter.
type Config struct {
	ChannelID      string
	SessionPath    string
	AllowedUserIDs map[string]bool
}

// Channel adapts WhatsApp events from whatsmeow into the app's normalized message model.
type Channel struct {
	cfg       Config
	client    *whatsmeow.Client
	container *sqlstore.Container
	publisher inbound.Publisher
	logger    *slog.Logger

	mu        sync.RWMutex
	rootCtx   context.Context
	cancel    context.CancelFunc
	started   bool
	currentQR string
}

// New creates a new WhatsApp channel adapter.
func New(cfg Config, publisher inbound.Publisher, logger *slog.Logger) (*Channel, error) {
	if strings.TrimSpace(cfg.SessionPath) == "" {
		return nil, errors.New("whatsapp session path is required")
	}
	if publisher == nil {
		return nil, errors.New("publisher is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Channel{
		cfg:       cfg,
		publisher: publisher,
		logger:    logger,
	}, nil
}

// Start initializes the whatsmeow client and connects to WhatsApp.
func (c *Channel) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return errors.New("whatsapp channel already started")
	}

	rootCtx, cancel := context.WithCancel(ctx)
	c.rootCtx = rootCtx
	c.cancel = cancel
	c.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(c.cfg.SessionPath), 0o700); err != nil {
		c.resetStartState()
		return fmt.Errorf("create whatsapp session directory: %w", err)
	}

	container, err := sqlstore.New(rootCtx, "sqlite3", "file:"+c.cfg.SessionPath+"?_foreign_keys=on", waLog.Stdout("whatsapp", "INFO", false))
	if err != nil {
		c.resetStartState()
		return fmt.Errorf("open whatsapp session store: %w", err)
	}

	device, err := container.GetFirstDevice(rootCtx)
	if err != nil {
		_ = container.Close()
		c.resetStartState()
		return fmt.Errorf("get whatsapp device: %w", err)
	}

	client := whatsmeow.NewClient(device, waLog.Stdout("whatsapp", "INFO", false))
	client.AddEventHandler(c.handleEvent)

	var qrChan <-chan whatsmeow.QRChannelItem
	if client.Store.ID == nil {
		qrChan, err = client.GetQRChannel(rootCtx)
		if err != nil {
			_ = container.Close()
			c.resetStartState()
			return fmt.Errorf("prepare whatsapp QR channel: %w", err)
		}
	}

	if err := client.Connect(); err != nil {
		_ = container.Close()
		c.resetStartState()
		return fmt.Errorf("connect whatsapp client: %w", err)
	}

	c.mu.Lock()
	c.client = client
	c.container = container
	c.started = true
	c.mu.Unlock()

	if qrChan != nil {
		go c.consumeQR(rootCtx, qrChan)
	}

	c.logger.Info("whatsapp connected", "session_path", c.cfg.SessionPath)
	return nil
}

// Stop disconnects the whatsmeow client.
func (c *Channel) Stop() error {
	c.mu.Lock()
	cancel := c.cancel
	client := c.client
	container := c.container
	c.cancel = nil
	c.rootCtx = nil
	c.client = nil
	c.container = nil
	c.started = false
	c.currentQR = ""
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if client != nil {
		client.Disconnect()
	}
	if container != nil {
		return container.Close()
	}
	return nil
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "whatsapp"
}

// QRCode returns the latest pending QR payload for linking the account.
func (c *Channel) QRCode() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentQR
}

// Send posts a text reply back to WhatsApp.
func (c *Channel) Send(ctx context.Context, msg *inbound.OutboundMessage) error {
	if msg == nil {
		return errors.New("outbound message is nil")
	}
	if strings.TrimSpace(msg.ChatID) == "" {
		return errors.New("outbound message ChatID is required")
	}

	c.mu.RLock()
	client := c.client
	started := c.started
	c.mu.RUnlock()
	if !started || client == nil {
		return errors.New("whatsapp channel is not running")
	}

	target, err := types.ParseJID(strings.TrimSpace(msg.ChatID))
	if err != nil {
		return fmt.Errorf("parse whatsapp chat id: %w", err)
	}

	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return errors.New("outbound message has no content")
	}

	_, err = client.SendMessage(ctx, target, &waE2E.Message{
		Conversation: proto.String(content),
	})
	if err != nil {
		return fmt.Errorf("send whatsapp message: %w", err)
	}
	return nil
}

func (c *Channel) handleEvent(evt any) {
	switch v := evt.(type) {
	case *events.Message:
		c.handleMessage(v)
	case *events.Connected:
		c.setCurrentQR("")
		c.logger.Info("whatsapp authenticated and connected")
	case *events.LoggedOut:
		c.setCurrentQR("")
		c.logger.Warn("whatsapp logged out", "reason", v.Reason.String())
	case *events.TemporaryBan:
		c.logger.Warn("whatsapp temporary ban", "details", v.String())
	}
}

func (c *Channel) handleMessage(evt *events.Message) {
	if evt == nil || evt.Message == nil {
		return
	}
	if evt.Info.IsFromMe {
		return
	}

	senderID := evt.Info.Sender.String()
	if senderID == "" {
		senderID = evt.Info.Chat.String()
	}
	if !c.isAllowedUser(senderID) {
		c.logger.Warn("whatsapp user not allowed", "sender_id", senderID)
		return
	}

	content := extractText(evt.Message)
	if content == "" {
		content = "[unsupported message]"
	}

	inboundMsg := &inbound.Message{
		Channel:   c.Name(),
		ChannelID: c.cfg.ChannelID,
		SenderID:  senderID,
		Sender:    strings.TrimSpace(evt.Info.PushName),
		ChatID:    evt.Info.Chat.String(),
		MessageID: string(evt.Info.ID),
		Content:   content,
		Metadata: map[string]any{
			"chat":       evt.Info.Chat.String(),
			"sender":     senderID,
			"is_group":   evt.Info.IsGroup,
			"push_name":  evt.Info.PushName,
			"message_ts": evt.Info.Timestamp,
		},
		Timestamp: evt.Info.Timestamp.UTC(),
	}

	ctx := c.runCtx()
	if err := c.publisher.PublishInbound(ctx, inboundMsg); err != nil {
		c.logger.Error("whatsapp publish inbound failed", "err", err)
	}
}

func (c *Channel) consumeQR(ctx context.Context, qrChan <-chan whatsmeow.QRChannelItem) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-qrChan:
			if !ok {
				return
			}
			switch evt.Event {
			case whatsmeow.QRChannelEventCode:
				c.setCurrentQR(evt.Code)
				c.logger.Info("whatsapp scan qr to link account", "code", evt.Code)
			case "success":
				c.setCurrentQR("")
				c.logger.Info("whatsapp pairing successful")
			default:
				if evt.Event != "" {
					c.setCurrentQR("")
				}
				if evt.Error != nil {
					c.logger.Warn("whatsapp pairing event", "event", evt.Event, "err", evt.Error)
				} else {
					c.logger.Info("whatsapp pairing event", "event", evt.Event)
				}
			}
		}
	}
}

func (c *Channel) resetStartState() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	c.cancel = nil
	c.rootCtx = nil
	c.client = nil
	c.container = nil
	c.started = false
	c.currentQR = ""
}

func (c *Channel) runCtx() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.rootCtx != nil {
		return c.rootCtx
	}
	return context.Background()
}

func (c *Channel) isAllowedUser(userID string) bool {
	if len(c.cfg.AllowedUserIDs) == 0 {
		return true
	}
	return c.cfg.AllowedUserIDs[strings.TrimSpace(userID)]
}

func (c *Channel) setCurrentQR(code string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentQR = code
}

func extractText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	switch {
	case strings.TrimSpace(msg.GetConversation()) != "":
		return strings.TrimSpace(msg.GetConversation())
	case msg.GetExtendedTextMessage() != nil && strings.TrimSpace(msg.GetExtendedTextMessage().GetText()) != "":
		return strings.TrimSpace(msg.GetExtendedTextMessage().GetText())
	case msg.GetImageMessage() != nil && strings.TrimSpace(msg.GetImageMessage().GetCaption()) != "":
		return strings.TrimSpace(msg.GetImageMessage().GetCaption())
	case msg.GetVideoMessage() != nil && strings.TrimSpace(msg.GetVideoMessage().GetCaption()) != "":
		return strings.TrimSpace(msg.GetVideoMessage().GetCaption())
	case msg.GetDocumentMessage() != nil && strings.TrimSpace(msg.GetDocumentMessage().GetCaption()) != "":
		return strings.TrimSpace(msg.GetDocumentMessage().GetCaption())
	default:
		return ""
	}
}
