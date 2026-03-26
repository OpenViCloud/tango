package rest

import (
	appservices "tango/internal/application/services"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

type DiscordRuntimeHandler struct {
	service appservices.DiscordRuntimeService
}

type discordRuntimeRequest struct {
	Token                      string   `json:"token"`
	RequireMention             bool     `json:"require_mention"`
	EnableTyping               bool     `json:"enable_typing"`
	EnableMessageContentIntent bool     `json:"enable_message_content_intent"`
	AllowedUserIDs             []string `json:"allowed_user_ids"`
}

type discordRuntimeResponse struct {
	Channel                    string   `json:"channel"`
	Running                    bool     `json:"running"`
	TokenConfigured            bool     `json:"token_configured"`
	RequireMention             bool     `json:"require_mention"`
	EnableTyping               bool     `json:"enable_typing"`
	EnableMessageContentIntent bool     `json:"enable_message_content_intent"`
	AllowedUserIDs             []string `json:"allowed_user_ids"`
}

func NewDiscordRuntimeHandler(service appservices.DiscordRuntimeService) *DiscordRuntimeHandler {
	return &DiscordRuntimeHandler{service: service}
}

func (h *DiscordRuntimeHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/runtime/discord/status", h.Status)
	rg.POST("/runtime/discord/start", h.Start)
	rg.POST("/runtime/discord/restart", h.Restart)
	rg.POST("/runtime/discord/stop", h.Stop)
}

// Status godoc
// @Summary Get Discord runtime status
// @Tags discord
// @Produce json
// @Security BearerAuth
// @Success 200 {object} discordRuntimeResponse
// @Failure 500 {object} errorResponse
// @Router /runtime/discord/status [get]
func (h *DiscordRuntimeHandler) Status(c *gin.Context) {
	response.OK(c, toDiscordRuntimeResponse(h.service.Status(c.Request.Context())))
}

// Start godoc
// @Summary Start or reload Discord runtime
// @Tags discord
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body discordRuntimeRequest true "Discord runtime payload"
// @Success 200 {object} discordRuntimeResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /runtime/discord/start [post]
func (h *DiscordRuntimeHandler) Start(c *gin.Context) {
	var req discordRuntimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}

	if err := h.service.Start(c.Request.Context(), appservices.DiscordRuntimeConfig{
		Token:                      req.Token,
		RequireMention:             req.RequireMention,
		EnableTyping:               req.EnableTyping,
		EnableMessageContentIntent: req.EnableMessageContentIntent,
		AllowedUserIDs:             req.AllowedUserIDs,
	}); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	response.OK(c, toDiscordRuntimeResponse(h.service.Status(c.Request.Context())))
}

// Restart godoc
// @Summary Restart Discord runtime with current config
// @Tags discord
// @Produce json
// @Security BearerAuth
// @Success 200 {object} discordRuntimeResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /runtime/discord/restart [post]
func (h *DiscordRuntimeHandler) Restart(c *gin.Context) {
	if err := h.service.Restart(c.Request.Context()); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	response.OK(c, toDiscordRuntimeResponse(h.service.Status(c.Request.Context())))
}

// Stop godoc
// @Summary Stop Discord runtime
// @Tags discord
// @Produce json
// @Security BearerAuth
// @Success 200 {object} discordRuntimeResponse
// @Failure 500 {object} errorResponse
// @Router /runtime/discord/stop [post]
func (h *DiscordRuntimeHandler) Stop(c *gin.Context) {
	if err := h.service.Stop(c.Request.Context()); err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	response.OK(c, toDiscordRuntimeResponse(h.service.Status(c.Request.Context())))
}

func toDiscordRuntimeResponse(status appservices.DiscordRuntimeStatus) discordRuntimeResponse {
	return discordRuntimeResponse{
		Channel:                    "discord",
		Running:                    status.Running,
		TokenConfigured:            status.TokenConfigured,
		RequireMention:             status.RequireMention,
		EnableTyping:               status.EnableTyping,
		EnableMessageContentIntent: status.EnableMessageContentIntent,
		AllowedUserIDs:             append([]string(nil), status.AllowedUserIDs...),
	}
}
