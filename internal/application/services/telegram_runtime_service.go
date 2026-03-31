package services

import "context"

type TelegramProjectEnvironment struct {
	ID   string
	Name string
}

type TelegramResourcePort struct {
	HostPort     int
	InternalPort int
	Proto        string
	Label        string
}

type TelegramResource struct {
	ID            string
	Name          string
	Type          string
	Status        string
	Image         string
	Tag           string
	EnvironmentID string
	ContainerID   string
	Ports         []TelegramResourcePort
}

type TelegramProject struct {
	ID           string
	Name         string
	Environments []TelegramProjectEnvironment
}

type TelegramProjectNavigator interface {
	ListProjects(ctx context.Context) ([]TelegramProject, error)
	ListEnvironmentResources(ctx context.Context, environmentID string) ([]TelegramResource, error)
	GetResource(ctx context.Context, resourceID string) (TelegramResource, error)
	StartResource(ctx context.Context, resourceID string) error
	StopResource(ctx context.Context, resourceID string) error
	RestartResource(ctx context.Context, resourceID string) error
}

type TelegramRuntimeConfig struct {
	ChannelID      string
	Token          string
	EnableTyping   bool
	AllowedUserIDs []string
}

type TelegramRuntimeStatus struct {
	Running         bool     `json:"running"`
	TokenConfigured bool     `json:"token_configured"`
	EnableTyping    bool     `json:"enable_typing"`
	AllowedUserIDs  []string `json:"allowed_user_ids"`
}

type TelegramRuntimeService interface {
	Status(ctx context.Context) TelegramRuntimeStatus
	Start(ctx context.Context, cfg TelegramRuntimeConfig) error
	Restart(ctx context.Context) error
	Stop(ctx context.Context) error
}
