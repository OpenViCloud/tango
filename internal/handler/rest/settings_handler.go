package rest

import (
	"context"
	"net/http"

	"tango/internal/domain"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	configRepo   domain.PlatformConfigRepository
	fileProvider domain.TraefikFileProvider
}

func NewSettingsHandler(configRepo domain.PlatformConfigRepository, fileProvider domain.TraefikFileProvider) *SettingsHandler {
	return &SettingsHandler{configRepo: configRepo, fileProvider: fileProvider}
}

func (h *SettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/settings", h.GetSettings)
	rg.PATCH("/settings", h.UpdateSettings)
}

type settingsResponse struct {
	PublicIP          string `json:"public_ip"`
	BaseDomain        string `json:"base_domain"`
	WildcardEnabled   bool   `json:"wildcard_enabled"`
	TraefikNetwork    string `json:"traefik_network"`
	CertResolver      string `json:"cert_resolver"`
	AppDomain         string `json:"app_domain"`
	AppTLSEnabled     bool   `json:"app_tls_enabled"`
	AppBackendURL     string `json:"app_backend_url"`
	ResourceMountRoot string `json:"resource_mount_root"`
}

type updateSettingsRequest struct {
	PublicIP          *string `json:"public_ip"`
	BaseDomain        *string `json:"base_domain"`
	WildcardEnabled   *bool   `json:"wildcard_enabled"`
	TraefikNetwork    *string `json:"traefik_network"`
	CertResolver      *string `json:"cert_resolver"`
	AppDomain         *string `json:"app_domain"`
	AppTLSEnabled     *bool   `json:"app_tls_enabled"`
	AppBackendURL     *string `json:"app_backend_url"`
	ResourceMountRoot *string `json:"resource_mount_root"`
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
		AppBackendURL:   "http://app:8080",
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
		case domain.PlatformConfigAppDomain:
			resp.AppDomain = cfg.Value
		case domain.PlatformConfigAppTLSEnabled:
			resp.AppTLSEnabled = cfg.Value == "true"
		case domain.PlatformConfigAppBackendURL:
			if cfg.Value != "" {
				resp.AppBackendURL = cfg.Value
			}
		case domain.PlatformConfigResourceMountRoot:
			resp.ResourceMountRoot = cfg.Value
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
	if req.AppDomain != nil {
		if err := h.configRepo.Set(ctx, domain.PlatformConfigAppDomain, *req.AppDomain); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.AppTLSEnabled != nil {
		val := "false"
		if *req.AppTLSEnabled {
			val = "true"
		}
		if err := h.configRepo.Set(ctx, domain.PlatformConfigAppTLSEnabled, val); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.AppBackendURL != nil {
		if err := h.configRepo.Set(ctx, domain.PlatformConfigAppBackendURL, *req.AppBackendURL); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.ResourceMountRoot != nil {
		if err := h.configRepo.Set(ctx, domain.PlatformConfigResourceMountRoot, *req.ResourceMountRoot); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Refresh app Traefik config whenever domain-related settings change
	if h.fileProvider != nil && (req.AppDomain != nil || req.AppTLSEnabled != nil || req.CertResolver != nil || req.AppBackendURL != nil) {
		h.refreshAppConfig(ctx)
	}

	h.GetSettings(c)
}

func (h *SettingsHandler) refreshAppConfig(ctx context.Context) {
	appDomain := ""
	appTLS := false
	certResolver := ""
	backendURL := "http://app:8080"

	if cfg, err := h.configRepo.Get(ctx, domain.PlatformConfigAppDomain); err == nil {
		appDomain = cfg.Value
	}
	if cfg, err := h.configRepo.Get(ctx, domain.PlatformConfigAppTLSEnabled); err == nil {
		appTLS = cfg.Value == "true"
	}
	if cfg, err := h.configRepo.Get(ctx, domain.PlatformConfigCertResolver); err == nil {
		certResolver = cfg.Value
	}
	if cfg, err := h.configRepo.Get(ctx, domain.PlatformConfigAppBackendURL); err == nil && cfg.Value != "" {
		backendURL = cfg.Value
	}

	_ = h.fileProvider.WriteAppConfig(appDomain, appTLS, certResolver, backendURL)
}
