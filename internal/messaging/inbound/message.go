package inbound

import (
	"context"
	"io"
	"time"
)

// MediaType represents a normalized attachment category across messaging channels.
type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeDocument MediaType = "document"
)

// Media describes an attachment extracted from an inbound message.
type Media struct {
	Type        MediaType
	URL         string
	Filename    string
	ContentType string
	Size        int
}

// Message is the normalized inbound message passed from channel adapters into the app.
type Message struct {
	Channel   string
	ChannelID string
	SenderID  string
	Sender    string
	ChatID    string
	GuildID   string
	MessageID string
	Content   string
	Media     []Media
	Metadata  map[string]any
	Timestamp time.Time
}

// OutboundFile represents a file to upload back through a messaging channel.
type OutboundFile struct {
	Name   string
	Reader io.Reader
}

// OutboundMessage is the normalized message a channel adapter can send outward.
type OutboundMessage struct {
	ChatID   string
	Content  string
	ReplyTo  string
	Files    []OutboundFile
	Typing   bool
	Metadata map[string]any
}

// Publisher consumes normalized inbound messages from any messaging channel.
type Publisher interface {
	PublishInbound(ctx context.Context, msg *Message) error
}

// Sender is the minimal outbound capability exposed to app-level publishers.
type Sender interface {
	Send(ctx context.Context, msg *OutboundMessage) error
}

// SenderAwarePublisher can receive the initialized channel sender during bootstrap.
type SenderAwarePublisher interface {
	Publisher
	SetSender(sender Sender)
}
