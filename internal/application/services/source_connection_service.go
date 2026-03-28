package services

import (
	"context"
	"time"
)

type SourceIntegrationState struct {
	UserID     string `json:"user_id"`
	Provider   string `json:"provider"`
	RedirectTo string `json:"redirect_to"`
	AppName    string `json:"app_name,omitempty"`
	SetupState string `json:"setup_state,omitempty"`
	ProviderID string `json:"provider_id,omitempty"`
}

type IntegrationStateStore interface {
	Save(ctx context.Context, state string, value SourceIntegrationState, ttl time.Duration) error
	Consume(ctx context.Context, state string) (*SourceIntegrationState, error)
}

type GitRepository struct {
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	FullName   string `json:"full_name"`
	CloneURL   string `json:"clone_url"`
	Private    bool   `json:"private"`
	DefaultRef string `json:"default_ref"`
}

type GitBranch struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

type GitHubAppManifest struct {
	Name               string               `json:"name"`
	URL                string               `json:"url"`
	RedirectURL        string               `json:"redirect_url"`
	CallbackURLs       []string             `json:"callback_urls,omitempty"`
	SetupURL           string               `json:"setup_url"`
	HookAttributes     GitHubHookAttributes `json:"hook_attributes"`
	Public             bool                 `json:"public"`
	DefaultEvents      []string             `json:"default_events,omitempty"`
	DefaultPermissions map[string]string    `json:"default_permissions"`
}

type GitHubHookAttributes struct {
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

type GitHubAppRegistration struct {
	AppID         int64
	ClientID      string
	ClientSecret  string
	WebhookSecret string
	PrivateKeyPEM string
	Name          string
	Slug          string
	HTMLURL       string
}

type GitHubAppCredentials struct {
	AppID         int64  `json:"app_id"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	WebhookSecret string `json:"webhook_secret"`
	PrivateKeyPEM string `json:"private_key_pem"`
}

type GitHubInstallation struct {
	ID           int64
	AccountLogin string
	AccountType  string
}

type GitHubAppService interface {
	BuildManifest(appName, redirectURL, setupURL, webhookURL string) GitHubAppManifest
	ExchangeManifest(ctx context.Context, code string) (*GitHubAppRegistration, error)
	BuildCreateAppURL(state string) string
	BuildInstallURL(appSlug, state string) string
	GetInstallation(ctx context.Context, credentials GitHubAppCredentials, installationID int64) (*GitHubInstallation, error)
	CreateInstallationToken(ctx context.Context, credentials GitHubAppCredentials, installationID int64) (string, error)
	ListRepositories(ctx context.Context, installationToken string) ([]GitRepository, error)
	ListBranches(ctx context.Context, installationToken, owner, repo string) ([]GitBranch, error)
}
