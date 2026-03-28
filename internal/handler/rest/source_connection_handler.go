package rest

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"tango/internal/application/command"
	"tango/internal/application/query"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

type SourceConnectionHandler struct {
	beginManifest    *command.BeginGitHubAppManifestHandler
	completeManifest *command.CompleteGitHubAppManifestHandler
	completeSetup    *command.CompleteGitHubAppSetupHandler
	resolveToken     *command.ResolveSourceConnectionTokenHandler
	listConnections  *query.ListSourceConnectionsHandler
	listRepos        *query.ListGitHubRepositoriesHandler
	listBranches     *query.ListGitHubBranchesHandler
	defaultRedirect  string
	apiBaseURL       string
}

func NewSourceConnectionHandler(
	beginManifest *command.BeginGitHubAppManifestHandler,
	completeManifest *command.CompleteGitHubAppManifestHandler,
	completeSetup *command.CompleteGitHubAppSetupHandler,
	resolveToken *command.ResolveSourceConnectionTokenHandler,
	listConnections *query.ListSourceConnectionsHandler,
	listRepos *query.ListGitHubRepositoriesHandler,
	listBranches *query.ListGitHubBranchesHandler,
	defaultRedirect string,
	apiBaseURL string,
) *SourceConnectionHandler {
	return &SourceConnectionHandler{
		beginManifest:    beginManifest,
		completeManifest: completeManifest,
		completeSetup:    completeSetup,
		resolveToken:     resolveToken,
		listConnections:  listConnections,
		listRepos:        listRepos,
		listBranches:     listBranches,
		defaultRedirect:  defaultRedirect,
		apiBaseURL:       apiBaseURL,
	}
}

func (h *SourceConnectionHandler) RegisterPublic(rg *gin.RouterGroup) {
	rg.GET("/source-connections/github/callback", h.CompleteGitHubAppManifest)
	rg.GET("/source-connections/github/setup", h.CompleteGitHubAppSetup)
	rg.POST("/source-connections/github/webhook", h.Webhook)
}

func (h *SourceConnectionHandler) RegisterProtected(rg *gin.RouterGroup) {
	rg.GET("/source-connections", h.List)
	rg.POST("/source-connections/github/apps", h.BeginGitHubAppManifest)
	rg.GET("/source-connections/:id/repos", h.ListGitHubRepos)
	rg.GET("/source-connections/:id/repos/:owner/:repo/branches", h.ListGitHubBranches)
}

type beginGitHubAppManifestRequest struct {
	AppName    string `json:"app_name" binding:"required"`
	RedirectTo string `json:"redirect_to"`
}

func (h *SourceConnectionHandler) List(c *gin.Context) {
	items, err := h.listConnections.Handle(c.Request.Context(), query.ListSourceConnectionsQuery{UserID: c.GetString("user_id")})
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *SourceConnectionHandler) BeginGitHubAppManifest(c *gin.Context) {
	var req beginGitHubAppManifestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	redirectTo := strings.TrimSpace(req.RedirectTo)
	if redirectTo == "" {
		redirectTo = h.defaultRedirect
	}
	result, err := h.beginManifest.Handle(c.Request.Context(), command.BeginGitHubAppManifestCommand{
		UserID:     c.GetString("user_id"),
		AppName:    req.AppName,
		RedirectTo: redirectTo,
		BaseURL:    h.apiBaseURL,
	})
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	slog.InfoContext(c.Request.Context(), "github app manifest prepared",
		"createUrl", result.CreateURL,
		"redirectUrl", result.Manifest.RedirectURL,
		"callbackUrls", result.Manifest.CallbackURLs,
		"setupUrl", result.Manifest.SetupURL,
		"webhookUrl", result.Manifest.HookAttributes.URL,
	)
	response.OK(c, result)
}

func (h *SourceConnectionHandler) CompleteGitHubAppManifest(c *gin.Context) {
	result, err := h.completeManifest.Handle(c.Request.Context(), command.CompleteGitHubAppManifestCommand{
		Code:  c.Query("code"),
		State: c.Query("state"),
	})
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	c.Redirect(http.StatusFound, result.InstallURL)
}

func (h *SourceConnectionHandler) CompleteGitHubAppSetup(c *gin.Context) {
	result, err := h.completeSetup.Handle(c.Request.Context(), command.CompleteGitHubAppSetupCommand{
		SetupState:     c.Query("setup_state"),
		InstallationID: c.Query("installation_id"),
	})
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	redirectTo := strings.TrimSpace(result.RedirectTo)
	if redirectTo == "" {
		redirectTo = h.defaultRedirect
	}
	if strings.Contains(redirectTo, "?") {
		redirectTo += "&github_connected=1"
	} else {
		redirectTo += "?github_connected=1"
	}
	c.Redirect(http.StatusFound, redirectTo)
}

func (h *SourceConnectionHandler) Webhook(c *gin.Context) {
	response.NoContent(c)
}

func (h *SourceConnectionHandler) ListGitHubRepos(c *gin.Context) {
	token, err := h.resolveToken.Handle(c.Request.Context(), c.GetString("user_id"), c.Param("id"))
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	items, err := h.listRepos.Handle(c.Request.Context(), query.ListGitHubRepositoriesQuery{AccessToken: token})
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *SourceConnectionHandler) ListGitHubBranches(c *gin.Context) {
	token, err := h.resolveToken.Handle(c.Request.Context(), c.GetString("user_id"), c.Param("id"))
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	items, err := h.listBranches.Handle(c.Request.Context(), query.ListGitHubBranchesQuery{
		AccessToken: token,
		Owner:       c.Param("owner"),
		Repo:        c.Param("repo"),
	})
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	response.OK(c, items)
}

func writeSourceConnectionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrSourceConnectionNotFound):
		_ = c.Error(response.NotFound(err.Error()))
	case errors.Is(err, domain.ErrSourceProviderNotFound):
		_ = c.Error(response.NotFound(err.Error()))
	case errors.Is(err, domain.ErrSourceConnectionCredentialsAbsent):
		_ = c.Error(response.BadRequest(err.Error()))
	case errors.Is(err, domain.ErrSourceConnectionOAuthStateInvalid):
		_ = c.Error(response.BadRequest(err.Error()))
	case errors.Is(err, domain.ErrSourceConnectionEncryptionFailed), errors.Is(err, domain.ErrSourceProviderEncryptionFailed):
		_ = c.Error(response.Internal(err.Error()))
	default:
		_ = c.Error(response.InternalCause(err, ""))
	}
}
