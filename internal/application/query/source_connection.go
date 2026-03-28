package query

import (
	"context"
	"encoding/json"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type SourceConnectionView struct {
	SourceProviderID  string         `json:"source_provider_id"`
	ID                string         `json:"id"`
	Provider          string         `json:"provider"`
	DisplayName       string         `json:"display_name"`
	AccountIdentifier string         `json:"account_identifier"`
	ExternalID        string         `json:"external_id"`
	Status            string         `json:"status"`
	ExpiresAt         string         `json:"expires_at,omitempty"`
	LastUsedAt        string         `json:"last_used_at,omitempty"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

type ListSourceConnectionsQuery struct {
	UserID string
}

type ListSourceConnectionsHandler struct {
	repo domain.SourceConnectionRepository
}

func NewListSourceConnectionsHandler(repo domain.SourceConnectionRepository) *ListSourceConnectionsHandler {
	return &ListSourceConnectionsHandler{repo: repo}
}

func (h *ListSourceConnectionsHandler) Handle(ctx context.Context, q ListSourceConnectionsQuery) ([]SourceConnectionView, error) {
	items, err := h.repo.ListByUser(ctx, strings.TrimSpace(q.UserID))
	if err != nil {
		return nil, err
	}
	result := make([]SourceConnectionView, 0, len(items))
	for _, item := range items {
		result = append(result, toSourceConnectionView(item))
	}
	return result, nil
}

type ListGitHubRepositoriesQuery struct {
	AccessToken string
}

type ListGitHubRepositoriesHandler struct {
	github appservices.GitHubAppService
}

func NewListGitHubRepositoriesHandler(github appservices.GitHubAppService) *ListGitHubRepositoriesHandler {
	return &ListGitHubRepositoriesHandler{github: github}
}

func (h *ListGitHubRepositoriesHandler) Handle(ctx context.Context, q ListGitHubRepositoriesQuery) ([]appservices.GitRepository, error) {
	return h.github.ListRepositories(ctx, strings.TrimSpace(q.AccessToken))
}

type ListGitHubBranchesQuery struct {
	AccessToken string
	Owner       string
	Repo        string
}

type ListGitHubBranchesHandler struct {
	github appservices.GitHubAppService
}

func NewListGitHubBranchesHandler(github appservices.GitHubAppService) *ListGitHubBranchesHandler {
	return &ListGitHubBranchesHandler{github: github}
}

func (h *ListGitHubBranchesHandler) Handle(ctx context.Context, q ListGitHubBranchesQuery) ([]appservices.GitBranch, error) {
	return h.github.ListBranches(ctx, strings.TrimSpace(q.AccessToken), strings.TrimSpace(q.Owner), strings.TrimSpace(q.Repo))
}

func toSourceConnectionView(connection *domain.SourceConnection) SourceConnectionView {
	metadata := map[string]any{}
	_ = json.Unmarshal([]byte(connection.MetadataJSON), &metadata)

	view := SourceConnectionView{
		SourceProviderID:  connection.SourceProviderID,
		ID:                connection.ID,
		Provider:          string(connection.Provider),
		DisplayName:       connection.DisplayName,
		AccountIdentifier: connection.AccountIdentifier,
		ExternalID:        connection.ExternalID,
		Status:            string(connection.Status),
		Metadata:          metadata,
		CreatedAt:         connection.CreatedAt.Format(timeLayout),
		UpdatedAt:         connection.UpdatedAt.Format(timeLayout),
	}
	if connection.ExpiresAt != nil {
		view.ExpiresAt = connection.ExpiresAt.Format(timeLayout)
	}
	if connection.LastUsedAt != nil {
		view.LastUsedAt = connection.LastUsedAt.Format(timeLayout)
	}
	return view
}

const timeLayout = "2006-01-02T15:04:05Z07:00"
