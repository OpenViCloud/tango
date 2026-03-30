package rest

import (
	"net/http"

	"tango/internal/domain"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	configRepo domain.PlatformConfigRepository
}

func NewSettingsHandler(configRepo domain.PlatformConfigRepository) *SettingsHandler {
	return &SettingsHandler{configRepo: configRepo}
}

func (h *SettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/settings", h.GetSettings)
	rg.PATCH("/settings", h.UpdateSettings)
}

type settingsResponse struct {
	PublicIP        string `json:"public_ip"`
	BaseDomain      string `json:"base_domain"`
	WildcardEnabled bool   `json:"wildcard_enabled"`
	TraefikNetwork  string `json:"traefik_network"`
	CertResolver    string `json:"cert_resolver"`
}

type updateSettingsRequest struct {
	PublicIP        *string `json:"public_ip"`
	BaseDomain      *string `json:"base_domain"`
	WildcardEnabled *bool   `json:"wildcard_enabled"`
	TraefikNetwork  *string `json:"traefik_network"`
	CertResolver    *string `json:"cert_resolver"`
}

func (h *SettingsHandler) GetSettings(c *gin.Context) {
	configs, err := h.configRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := settingsResponse{
		BaseDomain:      "localhost",
		WildcardEnabled: true,
	}
	for _, cfg := range configs {
		switch cfg.Key {
		case domain.PlatformConfigPublicIP:
			resp.PublicIP = cfg.Value
		case domain.PlatformConfigBaseDomain:
			resp.BaseDomain = cfg.Value
		case domain.PlatformConfigWildcardEnabled:
			resp.WildcardEnabled = cfg.Value == "true"
		case domain.PlatformConfigTraefikNetwork:
			resp.TraefikNetwork = cfg.Value
		case domain.PlatformConfigCertResolver:
			resp.CertResolver = cfg.Value
		}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var req updateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	if req.PublicIP != nil {
		if err := h.configRepo.Set(ctx, domain.PlatformConfigPublicIP, *req.PublicIP); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.BaseDomain != nil {
		if err := h.configRepo.Set(ctx, domain.PlatformConfigBaseDomain, *req.BaseDomain); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.WildcardEnabled != nil {
		val := "false"
		if *req.WildcardEnabled {
			val = "true"
		}
		if err := h.configRepo.Set(ctx, domain.PlatformConfigWildcardEnabled, val); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.TraefikNetwork != nil {
		if err := h.configRepo.Set(ctx, domain.PlatformConfigTraefikNetwork, *req.TraefikNetwork); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.CertResolver != nil {
		if err := h.configRepo.Set(ctx, domain.PlatformConfigCertResolver, *req.CertResolver); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	h.GetSettings(c)
}
