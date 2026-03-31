package rest

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"tango/internal/application/command"
	"tango/internal/application/query"
	appservices "tango/internal/application/services"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

type SourceConnectionHandler struct {
	beginManifest    *command.BeginGitHubAppManifestHandler
	completeManifest *command.CompleteGitHubAppManifestHandler
	completeSetup    *command.CompleteGitHubAppSetupHandler
	connectPAT       *command.ConnectPATHandler
	resolveToken     *command.ResolveSourceConnectionTokenHandler
	listConnections  *query.ListSourceConnectionsHandler
	listRepos        *query.ListGitHubRepositoriesHandler
	listUserRepos    *query.ListGitHubUserRepositoriesHandler
	listBranches     *query.ListGitHubBranchesHandler
	platformConfig   domain.PlatformConfigRepository
	defaultRedirect  string
	apiBaseURL       string
}

func NewSourceConnectionHandler(
	beginManifest *command.BeginGitHubAppManifestHandler,
	completeManifest *command.CompleteGitHubAppManifestHandler,
	completeSetup *command.CompleteGitHubAppSetupHandler,
	connectPAT *command.ConnectPATHandler,
	resolveToken *command.ResolveSourceConnectionTokenHandler,
	listConnections *query.ListSourceConnectionsHandler,
	listRepos *query.ListGitHubRepositoriesHandler,
	listUserRepos *query.ListGitHubUserRepositoriesHandler,
	listBranches *query.ListGitHubBranchesHandler,
	platformConfig domain.PlatformConfigRepository,
	defaultRedirect string,
	apiBaseURL string,
) *SourceConnectionHandler {
	return &SourceConnectionHandler{
		beginManifest:    beginManifest,
		completeManifest: completeManifest,
		completeSetup:    completeSetup,
		connectPAT:       connectPAT,
		resolveToken:     resolveToken,
		listConnections:  listConnections,
		listRepos:        listRepos,
		listUserRepos:    listUserRepos,
		listBranches:     listBranches,
		platformConfig:   platformConfig,
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
	rg.POST("/source-connections/pat", h.ConnectPAT)
	rg.GET("/source-connections/:id/repos", h.ListGitHubRepos)
	rg.GET("/source-connections/:id/repos/:owner/:repo/branches", h.ListGitHubBranches)
}

type beginGitHubAppManifestRequest struct {
	AppName    string `json:"app_name" binding:"required"`
	RedirectTo string `json:"redirect_to"`
}

type connectPATRequest struct {
	Token       string `json:"token"        binding:"required"`
	DisplayName string `json:"display_name"`
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
	publicBaseURL, err := h.resolveGitHubPublicBaseURL(c.Request.Context())
	if err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}
	result, err := h.beginManifest.Handle(c.Request.Context(), command.BeginGitHubAppManifestCommand{
		UserID:     c.GetString("user_id"),
		AppName:    req.AppName,
		RedirectTo: redirectTo,
		BaseURL:    publicBaseURL,
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

func (h *SourceConnectionHandler) ConnectPAT(c *gin.Context) {
	var req connectPATRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	connection, err := h.connectPAT.Handle(c.Request.Context(), command.ConnectPATCommand{
		UserID:      c.GetString("user_id"),
		Token:       req.Token,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	response.Created(c, connection)
}

func (h *SourceConnectionHandler) Webhook(c *gin.Context) {
	response.NoContent(c)
}

func (h *SourceConnectionHandler) ListGitHubRepos(c *gin.Context) {
	token, connType, err := h.resolveToken.Handle(c.Request.Context(), c.GetString("user_id"), c.Param("id"))
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	q := query.ListGitHubRepositoriesQuery{AccessToken: token}
	var items []appservices.GitRepository
	if connType == "github_pat" {
		items, err = h.listUserRepos.Handle(c.Request.Context(), q)
	} else {
		items, err = h.listRepos.Handle(c.Request.Context(), q)
	}
	if err != nil {
		writeSourceConnectionError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *SourceConnectionHandler) resolveGitHubPublicBaseURL(ctx context.Context) (string, error) {
	if h.platformConfig != nil {
		appDomain := ""
		appTLSEnabled := false

		if cfg, err := h.platformConfig.Get(ctx, domain.PlatformConfigAppDomain); err == nil {
			appDomain = normalizeHostLikeValue(cfg.Value)
		}
		if cfg, err := h.platformConfig.Get(ctx, domain.PlatformConfigAppTLSEnabled); err == nil {
			appTLSEnabled = strings.EqualFold(strings.TrimSpace(cfg.Value), "true")
		}
		if appDomain != "" {
			if !appTLSEnabled {
				return "", errors.New("GitHub App requires HTTPS. Enable App HTTPS in Settings before connecting GitHub.")
			}
			return "https://" + appDomain, nil
		}
	}

	baseURL := strings.TrimSpace(h.apiBaseURL)
	if strings.HasPrefix(strings.ToLower(baseURL), "https://") {
		return strings.TrimRight(baseURL, "/"), nil
	}

	return "", errors.New("GitHub App requires a public HTTPS URL. Configure App Domain and enable HTTPS first.")
}

func normalizeHostLikeValue(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	return strings.TrimRight(trimmed, "/")
}

func (h *SourceConnectionHandler) ListGitHubBranches(c *gin.Context) {
	token, _, err := h.resolveToken.Handle(c.Request.Context(), c.GetString("user_id"), c.Param("id"))
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
	case domain.IsUserFacing(err):
		var ufErr *domain.UserFacingError
		errors.As(err, &ufErr)
		_ = c.Error(response.BadRequest(ufErr.Error()))
	default:
		_ = c.Error(response.InternalCause(err, ""))
	}
}
