package rest

import (
	"errors"

	"tango/internal/application/command"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

// TunnelHandler exposes Cloudflare tunnel management endpoints.
// Routes are nested under /clusters/:id/tunnels.
type TunnelHandler struct {
	expose         *command.ExposeServiceHandler
	unexpose       *command.UnexposeServiceHandler
	repo           domain.ClusterTunnelRepository
	connectionRepo domain.CloudflareConnectionRepository
}

func NewTunnelHandler(
	expose *command.ExposeServiceHandler,
	unexpose *command.UnexposeServiceHandler,
	repo domain.ClusterTunnelRepository,
	connectionRepo domain.CloudflareConnectionRepository,
) *TunnelHandler {
	return &TunnelHandler{
		expose:         expose,
		unexpose:       unexpose,
		repo:           repo,
		connectionRepo: connectionRepo,
	}
}

func (h *TunnelHandler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/clusters/:id/tunnels")
	g.GET("", h.GetTunnel)
	g.POST("/import", h.ImportTunnel)
	g.POST("/expose", h.ExposeService)
	g.DELETE("/expose", h.UnexposeService)
}

// GetTunnel returns the cluster tunnel and its current exposures.
func (h *TunnelHandler) GetTunnel(c *gin.Context) {
	clusterID := c.Param("id")
	tunnel, err := h.repo.GetByClusterID(c.Request.Context(), clusterID)
	if err != nil {
		if errors.Is(err, domain.ErrClusterTunnelNotFound) {
			response.OK(c, (*tunnelResponse)(nil))
			return
		}
		_ = c.Error(response.Internal("get tunnel failed: " + err.Error()))
		return
	}
	if tunnel.CloudflareConnectionID != "" {
		connection, err := h.connectionRepo.GetByID(c.Request.Context(), tunnel.CloudflareConnectionID)
		if err != nil {
			_ = c.Error(response.Internal("get tunnel connection failed: " + err.Error()))
			return
		}
		if connection.UserID != c.GetString("user_id") {
			response.OK(c, (*tunnelResponse)(nil))
			return
		}
	}
	r := toTunnelResponse(tunnel)
	response.OK(c, r)
}

// ExposeService creates/updates the cluster tunnel to expose a new hostname.
func (h *TunnelHandler) ExposeService(c *gin.Context) {
	clusterID := c.Param("id")

	var req struct {
		ConnectionID string `json:"connection_id"`
		Hostname     string `json:"hostname"    binding:"required"`
		ServiceURL   string `json:"service_url" binding:"required"`
		Namespace    string `json:"namespace"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	tunnel, err := h.expose.Handle(c.Request.Context(), command.ExposeServiceCommand{
		UserID:       c.GetString("user_id"),
		ClusterID:    clusterID,
		ConnectionID: req.ConnectionID,
		Hostname:     req.Hostname,
		ServiceURL:   req.ServiceURL,
		Namespace:    req.Namespace,
	})
	if err != nil {
		writeTunnelError(c, "expose service failed: ", err)
		return
	}
	response.OK(c, toTunnelResponse(tunnel))
}

// ImportTunnel binds an existing Cloudflare tunnel to the cluster and deploys cloudflared.
func (h *TunnelHandler) ImportTunnel(c *gin.Context) {
	clusterID := c.Param("id")

	var req struct {
		ConnectionID string `json:"connection_id"`
		TunnelID     string `json:"tunnel_id" binding:"required"`
		TunnelToken  string `json:"tunnel_token" binding:"required"`
		Namespace    string `json:"namespace"`
		Overwrite    bool   `json:"overwrite"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	tunnel, err := h.expose.Import(c.Request.Context(), command.ImportClusterTunnelCommand{
		UserID:       c.GetString("user_id"),
		ClusterID:    clusterID,
		ConnectionID: req.ConnectionID,
		TunnelID:     req.TunnelID,
		TunnelToken:  req.TunnelToken,
		Namespace:    req.Namespace,
		Overwrite:    req.Overwrite,
	})
	if err != nil {
		writeTunnelError(c, "import tunnel failed: ", err)
		return
	}
	response.Created(c, toTunnelResponse(tunnel))
}

// UnexposeService removes a hostname from the cluster tunnel.
func (h *TunnelHandler) UnexposeService(c *gin.Context) {
	clusterID := c.Param("id")

	var req struct {
		Hostname string `json:"hostname" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	if err := h.unexpose.Handle(c.Request.Context(), command.UnexposeServiceCommand{
		UserID:    c.GetString("user_id"),
		ClusterID: clusterID,
		Hostname:  req.Hostname,
	}); err != nil {
		writeTunnelError(c, "unexpose service failed: ", err)
		return
	}
	response.NoContent(c)
}

// ── response shape ────────────────────────────────────────────────────────────

type tunnelExposureResponse struct {
	ID         string `json:"id"`
	Hostname   string `json:"hostname"`
	ServiceURL string `json:"service_url"`
}

type tunnelResponse struct {
	ID           string                   `json:"id"`
	ClusterID    string                   `json:"cluster_id"`
	ConnectionID string                   `json:"connection_id"`
	TunnelID     string                   `json:"tunnel_id"`
	Namespace    string                   `json:"namespace"`
	Exposures    []tunnelExposureResponse `json:"exposures"`
}

func toTunnelResponse(t *domain.ClusterTunnel) tunnelResponse {
	exps := make([]tunnelExposureResponse, 0, len(t.Exposures))
	for _, e := range t.Exposures {
		exps = append(exps, tunnelExposureResponse{
			ID:         e.ID,
			Hostname:   e.Hostname,
			ServiceURL: e.ServiceURL,
		})
	}
	return tunnelResponse{
		ID:           t.ID,
		ClusterID:    t.ClusterID,
		ConnectionID: t.CloudflareConnectionID,
		TunnelID:     t.TunnelID,
		Namespace:    t.Namespace,
		Exposures:    exps,
	}
}

func writeTunnelError(c *gin.Context, prefix string, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		_ = c.Error(response.BadRequest(prefix + err.Error()))
	case errors.Is(err, domain.ErrClusterTunnelAlreadyExists):
		_ = c.Error(response.BadRequest(prefix + err.Error()))
	case errors.Is(err, domain.ErrClusterTunnelConnectionRequired):
		_ = c.Error(response.BadRequest(prefix + err.Error()))
	case errors.Is(err, domain.ErrCloudflareConnectionNotFound), errors.Is(err, domain.ErrClusterTunnelNotFound):
		_ = c.Error(response.NotFound(prefix + err.Error()))
	default:
		_ = c.Error(response.Internal(prefix + err.Error()))
	}
}
