package inbound

import (
	"context"
	"log/slog"
	"sync"
)

// Service is a minimal inbound message handler used by the app composition root.
type Service struct {
	mu     sync.RWMutex
	logger *slog.Logger
	sender Sender
	senders map[string]Sender
}

// NewService creates a new inbound messaging service.
func NewService(logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{logger: logger}
}

// SetSender injects the outbound sender exposed by a messaging channel.
func (s *Service) SetSender(sender Sender) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sender = sender
	if named, ok := sender.(interface{ Name() string }); ok {
		if s.senders == nil {
			s.senders = make(map[string]Sender)
		}
		s.senders[named.Name()] = sender
	}
}

// PublishInbound logs inbound messages and sends a simple acknowledgement.
func (s *Service) PublishInbound(ctx context.Context, msg *Message) error {
	s.logger.InfoContext(ctx, "inbound message",
		"channel", msg.Channel,
		"chat_id", msg.ChatID,
		"sender", msg.Sender,
		"content", msg.Content,
	)

	s.mu.RLock()
	sender := s.sender
	if msg.Channel != "" && s.senders != nil {
		if namedSender, ok := s.senders[msg.Channel]; ok {
			sender = namedSender
		}
	}
	s.mu.RUnlock()

	if sender == nil {
		return nil
	}

	return sender.Send(ctx, &OutboundMessage{
		ChatID:  msg.ChatID,
		Content: "Bot received your message.",
		ReplyTo: msg.MessageID,
		Typing:  false,
	})
}
