package rest

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"tango/internal/domain"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	configRepo       domain.PlatformConfigRepository
	fileProvider     domain.TraefikFileProvider
	traefikRestarter domain.TraefikRestarter // optional, nil = no restart capability
}

func NewSettingsHandler(configRepo domain.PlatformConfigRepository, fileProvider domain.TraefikFileProvider, traefikRestarter domain.TraefikRestarter) *SettingsHandler {
	return &SettingsHandler{
		configRepo:       configRepo,
		fileProvider:     fileProvider,
		traefikRestarter: traefikRestarter,
	}
}

func (h *SettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/settings", h.GetSettings)
	rg.PATCH("/settings", h.UpdateSettings)
	rg.POST("/settings/traefik/restart", h.RestartTraefik)
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
	ACMEEmail         string `json:"acme_email"`
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
	ACMEEmail         *string `json:"acme_email"`
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
		TraefikNetwork:  "tango_net",
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
		case domain.PlatformConfigACMEEmail:
			resp.ACMEEmail = cfg.Value
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
	if req.ACMEEmail != nil {
		if err := h.configRepo.Set(ctx, domain.PlatformConfigACMEEmail, *req.ACMEEmail); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Refresh dynamic app Traefik config whenever domain-related settings change.
	if h.fileProvider != nil && (req.AppDomain != nil || req.AppTLSEnabled != nil || req.CertResolver != nil || req.AppBackendURL != nil) {
		h.refreshAppConfig(ctx)
	}

	// Rewrite static Traefik config when ACME email changes (no restart — user triggers that separately).
	if h.fileProvider != nil && req.ACMEEmail != nil {
		if err := h.fileProvider.WriteStaticConfig(*req.ACMEEmail); err != nil {
			slog.Warn("write traefik static config failed", "err", err)
		}
	}

	h.GetSettings(c)
}

// RestartTraefik rewrites the Traefik static config from DB and restarts the container.
func (h *SettingsHandler) RestartTraefik(c *gin.Context) {
	if h.fileProvider == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "traefik file provider not configured"})
		return
	}
	if h.traefikRestarter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "traefik restarter not available"})
		return
	}

	reqCtx := c.Request.Context()

	acmeEmail := ""
	if cfg, err := h.configRepo.Get(reqCtx, domain.PlatformConfigACMEEmail); err == nil {
		acmeEmail = cfg.Value
	}

	if err := h.fileProvider.WriteStaticConfig(acmeEmail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "write traefik config: " + err.Error()})
		return
	}

	// Use a detached context so the Docker restart is not cancelled if the
	// HTTP request times out or the client disconnects mid-restart.
	restartCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.traefikRestarter.RestartTraefik(restartCtx); err != nil {
		slog.Error("restart traefik failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "restart traefik: " + err.Error()})
		return
	}

	slog.Info("traefik restarted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "traefik restarted"})
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

