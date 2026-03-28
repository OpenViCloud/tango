package command

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type BeginGitHubAppManifestCommand struct {
	UserID     string
	AppName    string
	RedirectTo string
	BaseURL    string
}

type BeginGitHubAppManifestResult struct {
	CreateURL string                        `json:"create_url"`
	Manifest  appservices.GitHubAppManifest `json:"manifest"`
}

type BeginGitHubAppManifestHandler struct {
	stateStore appservices.IntegrationStateStore
	github     appservices.GitHubAppService
}

func NewBeginGitHubAppManifestHandler(stateStore appservices.IntegrationStateStore, github appservices.GitHubAppService) *BeginGitHubAppManifestHandler {
	return &BeginGitHubAppManifestHandler{stateStore: stateStore, github: github}
}

func (h *BeginGitHubAppManifestHandler) Handle(ctx context.Context, cmd BeginGitHubAppManifestCommand) (*BeginGitHubAppManifestResult, error) {
	manifestState := newRandomState("ghm_")
	setupState := newRandomState("ghs_")
	if err := h.stateStore.Save(ctx, manifestState, appservices.SourceIntegrationState{
		UserID:     strings.TrimSpace(cmd.UserID),
		Provider:   string(domain.SourceConnectionProviderGitHub),
		RedirectTo: strings.TrimSpace(cmd.RedirectTo),
		AppName:    strings.TrimSpace(cmd.AppName),
		SetupState: setupState,
	}, 30*time.Minute); err != nil {
		return nil, err
	}
	manifest := h.github.BuildManifest(
		strings.TrimSpace(cmd.AppName),
		strings.TrimRight(strings.TrimSpace(cmd.BaseURL), "/"),
		strings.TrimRight(strings.TrimSpace(cmd.BaseURL), "/")+"/api/source-connections/github/setup?setup_state="+setupState,
		strings.TrimRight(strings.TrimSpace(cmd.BaseURL), "/")+"/api/source-connections/github/webhook",
	)
	return &BeginGitHubAppManifestResult{
		CreateURL: h.github.BuildCreateAppURL(manifestState),
		Manifest:  manifest,
	}, nil
}

type CompleteGitHubAppManifestCommand struct {
	Code  string
	State string
}

type CompleteGitHubAppManifestResult struct {
	Provider   *domain.SourceProvider
	InstallURL string
}

type CompleteGitHubAppManifestHandler struct {
	providerRepo domain.SourceProviderRepository
	stateStore   appservices.IntegrationStateStore
	github       appservices.GitHubAppService
	cipher       appservices.SecretCipher
}

func NewCompleteGitHubAppManifestHandler(
	providerRepo domain.SourceProviderRepository,
	stateStore appservices.IntegrationStateStore,
	github appservices.GitHubAppService,
	cipher appservices.SecretCipher,
) *CompleteGitHubAppManifestHandler {
	return &CompleteGitHubAppManifestHandler{
		providerRepo: providerRepo,
		stateStore:   stateStore,
		github:       github,
		cipher:       cipher,
	}
}

func (h *CompleteGitHubAppManifestHandler) Handle(ctx context.Context, cmd CompleteGitHubAppManifestCommand) (*CompleteGitHubAppManifestResult, error) {
	stateValue, err := h.stateStore.Consume(ctx, strings.TrimSpace(cmd.State))
	if err != nil {
		return nil, domain.ErrSourceConnectionOAuthStateInvalid
	}
	registration, err := h.github.ExchangeManifest(ctx, strings.TrimSpace(cmd.Code))
	if err != nil {
		return nil, err
	}
	credentialsJSON, err := json.Marshal(appservices.GitHubAppCredentials{
		AppID:         registration.AppID,
		ClientID:      registration.ClientID,
		ClientSecret:  registration.ClientSecret,
		WebhookSecret: registration.WebhookSecret,
		PrivateKeyPEM: registration.PrivateKeyPEM,
	})
	if err != nil {
		return nil, err
	}
	encrypted, err := h.cipher.Encrypt(ctx, string(credentialsJSON))
	if err != nil {
		return nil, domain.ErrSourceProviderEncryptionFailed
	}
	metadataJSON, err := json.Marshal(map[string]any{
		"slug":     registration.Slug,
		"html_url": registration.HTMLURL,
		"app_id":   registration.AppID,
	})
	if err != nil {
		return nil, err
	}
	provider, err := domain.NewSourceProvider(
		newSourceProviderID(),
		stateValue.UserID,
		string(domain.SourceConnectionProviderGitHub),
		registration.Name,
		encrypted,
		string(metadataJSON),
		string(domain.SourceProviderStatusActive),
	)
	if err != nil {
		return nil, err
	}
	savedProvider, err := h.providerRepo.Save(ctx, provider)
	if err != nil {
		return nil, err
	}
	if err := h.stateStore.Save(ctx, stateValue.SetupState, appservices.SourceIntegrationState{
		UserID:     stateValue.UserID,
		Provider:   string(domain.SourceConnectionProviderGitHub),
		RedirectTo: stateValue.RedirectTo,
		ProviderID: savedProvider.ID,
	}, 2*time.Hour); err != nil {
		return nil, err
	}
	return &CompleteGitHubAppManifestResult{
		Provider:   savedProvider,
		InstallURL: h.github.BuildInstallURL(registration.Slug, stateValue.SetupState),
	}, nil
}

type CompleteGitHubAppSetupCommand struct {
	SetupState     string
	InstallationID string
}

type CompleteGitHubAppSetupResult struct {
	Connection *domain.SourceConnection
	RedirectTo string
}

type CompleteGitHubAppSetupHandler struct {
	providerRepo   domain.SourceProviderRepository
	connectionRepo domain.SourceConnectionRepository
	stateStore     appservices.IntegrationStateStore
	github         appservices.GitHubAppService
	cipher         appservices.SecretCipher
}

func NewCompleteGitHubAppSetupHandler(
	providerRepo domain.SourceProviderRepository,
	connectionRepo domain.SourceConnectionRepository,
	stateStore appservices.IntegrationStateStore,
	github appservices.GitHubAppService,
	cipher appservices.SecretCipher,
) *CompleteGitHubAppSetupHandler {
	return &CompleteGitHubAppSetupHandler{
		providerRepo:   providerRepo,
		connectionRepo: connectionRepo,
		stateStore:     stateStore,
		github:         github,
		cipher:         cipher,
	}
}

func (h *CompleteGitHubAppSetupHandler) Handle(ctx context.Context, cmd CompleteGitHubAppSetupCommand) (*CompleteGitHubAppSetupResult, error) {
	stateValue, err := h.stateStore.Consume(ctx, strings.TrimSpace(cmd.SetupState))
	if err != nil {
		return nil, domain.ErrSourceConnectionOAuthStateInvalid
	}
	provider, err := h.providerRepo.GetByID(ctx, stateValue.ProviderID)
	if err != nil {
		return nil, err
	}
	if provider.UserID != stateValue.UserID {
		return nil, domain.ErrSourceProviderNotFound
	}
	credentials, err := decryptGitHubAppCredentials(ctx, h.cipher, provider)
	if err != nil {
		return nil, err
	}
	installationID, err := strconv.ParseInt(strings.TrimSpace(cmd.InstallationID), 10, 64)
	if err != nil || installationID <= 0 {
		return nil, domain.ErrInvalidInput
	}
	installation, err := h.github.GetInstallation(ctx, credentials, installationID)
	if err != nil {
		return nil, err
	}
	metadataJSON, err := json.Marshal(map[string]any{
		"account_type": installation.AccountType,
	})
	if err != nil {
		return nil, err
	}
	connection, err := domain.NewSourceConnection(
		newSourceConnectionID(),
		stateValue.UserID,
		string(domain.SourceConnectionProviderGitHub),
		provider.ID,
		installation.AccountLogin,
		installation.AccountLogin,
		strconv.FormatInt(installation.ID, 10),
		string(metadataJSON),
		string(domain.SourceConnectionStatusActive),
		nil,
	)
	if err != nil {
		return nil, err
	}
	savedConnection, err := h.connectionRepo.Save(ctx, connection)
	if err != nil {
		return nil, err
	}
	return &CompleteGitHubAppSetupResult{
		Connection: savedConnection,
		RedirectTo: stateValue.RedirectTo,
	}, nil
}

type ResolveSourceConnectionTokenHandler struct {
	connectionRepo domain.SourceConnectionRepository
	providerRepo   domain.SourceProviderRepository
	cipher         appservices.SecretCipher
	github         appservices.GitHubAppService
}

func NewResolveSourceConnectionTokenHandler(
	connectionRepo domain.SourceConnectionRepository,
	providerRepo domain.SourceProviderRepository,
	cipher appservices.SecretCipher,
	github appservices.GitHubAppService,
) *ResolveSourceConnectionTokenHandler {
	return &ResolveSourceConnectionTokenHandler{
		connectionRepo: connectionRepo,
		providerRepo:   providerRepo,
		cipher:         cipher,
		github:         github,
	}
}

func (h *ResolveSourceConnectionTokenHandler) Handle(ctx context.Context, userID, connectionID string) (string, error) {
	connection, err := h.connectionRepo.GetByID(ctx, strings.TrimSpace(connectionID))
	if err != nil {
		return "", err
	}
	if connection.UserID != strings.TrimSpace(userID) {
		return "", domain.ErrSourceConnectionNotFound
	}
	provider, err := h.providerRepo.GetByID(ctx, connection.SourceProviderID)
	if err != nil {
		return "", err
	}
	if provider.UserID != connection.UserID {
		return "", domain.ErrSourceProviderNotFound
	}
	credentials, err := decryptGitHubAppCredentials(ctx, h.cipher, provider)
	if err != nil {
		return "", err
	}
	installationID, err := strconv.ParseInt(strings.TrimSpace(connection.ExternalID), 10, 64)
	if err != nil || installationID <= 0 {
		return "", domain.ErrSourceConnectionCredentialsAbsent
	}
	token, err := h.github.CreateInstallationToken(ctx, credentials, installationID)
	if err != nil {
		return "", err
	}
	_ = h.connectionRepo.TouchUsedAt(ctx, connection.ID, time.Now().UTC())
	return token, nil
}

func decryptGitHubAppCredentials(ctx context.Context, cipher appservices.SecretCipher, provider *domain.SourceProvider) (appservices.GitHubAppCredentials, error) {
	var credentials appservices.GitHubAppCredentials
	raw, err := cipher.Decrypt(ctx, provider.EncryptedCredentials)
	if err != nil {
		return credentials, domain.ErrSourceProviderEncryptionFailed
	}
	if err := json.Unmarshal([]byte(raw), &credentials); err != nil {
		return credentials, domain.ErrSourceProviderEncryptionFailed
	}
	return credentials, nil
}

func newSourceProviderID() string {
	return newRandomState("srcp_")
}

func newSourceConnectionID() string {
	return newRandomState("src_")
}

func newRandomState(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	}
	return prefix + hex.EncodeToString(b[:])
}
