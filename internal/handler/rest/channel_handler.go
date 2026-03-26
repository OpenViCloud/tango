package rest

import (
	"encoding/json"
	"errors"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/contract/common"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

type ChannelHandler struct {
	service        appservices.ChannelService
	runtimeService appservices.ChannelRuntimeService
}

type createChannelRequest struct {
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	Credentials json.RawMessage `json:"credentials"`
	Settings    json.RawMessage `json:"settings"`
}

type testChannelConnectionRequest struct {
	Kind        string          `json:"kind"`
	Credentials json.RawMessage `json:"credentials"`
	Settings    json.RawMessage `json:"settings"`
}

type testChannelConnectionRequestDoc struct {
	Kind        string         `json:"kind"`
	Credentials map[string]any `json:"credentials"`
	Settings    map[string]any `json:"settings"`
}

type channelTestConnectionResponse struct {
	Kind    string         `json:"kind"`
	OK      bool           `json:"ok"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type updateChannelRequest struct {
	Name               string          `json:"name"`
	Kind               string          `json:"kind"`
	Status             string          `json:"status"`
	Credentials        json.RawMessage `json:"credentials"`
	ReplaceCredentials bool            `json:"replace_credentials"`
	Settings           json.RawMessage `json:"settings"`
}

type channelQRCodeResponse struct {
	ID     string `json:"id"`
	QRCode string `json:"qr_code"`
}

type channelOperationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type channelListResponse struct {
	Items      []channelOperationResponse `json:"items"`
	PageIndex  int                        `json:"pageIndex"`
	PageSize   int                        `json:"pageSize"`
	TotalItems int64                      `json:"totalItems"`
	TotalPage  int                        `json:"totalPage"`
}

func NewChannelHandler(service appservices.ChannelService, runtimeService appservices.ChannelRuntimeService) *ChannelHandler {
	return &ChannelHandler{service: service, runtimeService: runtimeService}
}

func (h *ChannelHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/channels", h.List)
	rg.GET("/channels/:id", h.GetByID)
	rg.POST("/channels/test-connection", h.TestConnection)
	rg.POST("/channels", h.Create)
	rg.PUT("/channels/:id", h.Update)
	rg.DELETE("/channels/:id", h.Delete)
	rg.POST("/channels/:id/start", h.Start)
	rg.POST("/channels/:id/stop", h.Stop)
	rg.POST("/channels/:id/restart", h.Restart)
	rg.GET("/channels/:id/qr-code", h.QRCode)
}

// List godoc
// @Summary List channels
// @Tags channels
// @Produce json
// @Security BearerAuth
// @Success 200 {object} channelListResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /channels [get]
func (h *ChannelHandler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context(), common.BaseRequestModel{
		PageIndex:  parseIntDefault(c.Query("pageIndex"), 0),
		PageSize:   parseIntDefault(c.Query("pageSize"), 20),
		SearchText: c.Query("searchText"),
		OrderBy:    c.Query("orderBy"),
		Ascending:  c.Query("ascending") == "true",
	})
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, result)
}

// GetByID godoc
// @Summary Get channel by ID
// @Tags channels
// @Produce json
// @Security BearerAuth
// @Param id path string true "Channel ID"
// @Success 200 {object} channelOperationResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /channels/{id} [get]
func (h *ChannelHandler) GetByID(c *gin.Context) {
	view, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, view)
}

// TestConnection godoc
// @Summary Test channel connection
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body testChannelConnectionRequestDoc true "Channel connection payload"
// @Success 200 {object} channelTestConnectionResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /channels/test-connection [post]
func (h *ChannelHandler) TestConnection(c *gin.Context) {
	var req testChannelConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	if validationErr := validateTestChannelConnectionRequest(req); validationErr != nil {
		_ = c.Error(validationErr)
		return
	}

	view, err := h.service.TestConnection(c.Request.Context(), appservices.TestChannelConnectionInput{
		Kind:        req.Kind,
		Credentials: req.Credentials,
		Settings:    req.Settings,
	})
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, view)
}

// Create godoc
// @Summary Create channel
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createChannelRequest true "Channel payload"
// @Success 201 {object} channelOperationResponse
// @Failure 400 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /channels [post]
func (h *ChannelHandler) Create(c *gin.Context) {
	var req createChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	if validationErr := validateCreateChannelRequest(req); validationErr != nil {
		_ = c.Error(validationErr)
		return
	}

	view, err := h.service.Create(c.Request.Context(), appservices.CreateChannelInput{
		Name:        req.Name,
		Kind:        req.Kind,
		Status:      "pending",
		Credentials: req.Credentials,
		Settings:    req.Settings,
	})
	if err != nil {
		writeChannelError(c, err)
		return
	}

	// Auto-start if channel has credentials
	if view.HasCredentials {
		if _, startErr := h.runtimeService.Start(c.Request.Context(), view.ID); startErr != nil {
			writeChannelError(c, startErr)
			return
		}
		view.Status = string(domain.ChannelStatusActive)
	}

	response.Created(c, view)
}

// Update godoc
// @Summary Update channel
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Channel ID"
// @Param request body updateChannelRequest true "Channel payload"
// @Success 200 {object} channelOperationResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /channels/{id} [put]
func (h *ChannelHandler) Update(c *gin.Context) {
	var req updateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	if validationErr := validateUpdateChannelRequest(req); validationErr != nil {
		_ = c.Error(validationErr)
		return
	}

	view, err := h.service.Update(c.Request.Context(), appservices.UpdateChannelInput{
		ID:                 c.Param("id"),
		Name:               req.Name,
		Kind:               req.Kind,
		Status:             req.Status,
		Credentials:        req.Credentials,
		ReplaceCredentials: req.ReplaceCredentials,
		Settings:           req.Settings,
	})
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, view)
}

// Delete godoc
// @Summary Delete channel
// @Tags channels
// @Produce json
// @Security BearerAuth
// @Param id path string true "Channel ID"
// @Success 204
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /channels/{id} [delete]
func (h *ChannelHandler) Delete(c *gin.Context) {
	if _, err := h.runtimeService.Stop(c.Request.Context(), c.Param("id")); err != nil {
		writeChannelError(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeChannelError(c, err)
		return
	}
	response.NoContent(c)
}

// Start godoc
// @Summary Start channel runtime
// @Tags channels
// @Produce json
// @Security BearerAuth
// @Param id path string true "Channel ID"
// @Success 200 {object} channelOperationResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /channels/{id}/start [post]
func (h *ChannelHandler) Start(c *gin.Context) {
	view, err := h.runtimeService.Start(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, view)
}

// Stop godoc
// @Summary Stop channel runtime
// @Tags channels
// @Produce json
// @Security BearerAuth
// @Param id path string true "Channel ID"
// @Success 200 {object} channelOperationResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /channels/{id}/stop [post]
func (h *ChannelHandler) Stop(c *gin.Context) {
	view, err := h.runtimeService.Stop(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, view)
}

// Restart godoc
// @Summary Restart channel runtime
// @Tags channels
// @Produce json
// @Security BearerAuth
// @Param id path string true "Channel ID"
// @Success 200 {object} channelOperationResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /channels/{id}/restart [post]
func (h *ChannelHandler) Restart(c *gin.Context) {
	view, err := h.runtimeService.Restart(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, view)
}

// QRCode godoc
// @Summary Get channel QR code
// @Tags channels
// @Produce json
// @Security BearerAuth
// @Param id path string true "Channel ID"
// @Success 200 {object} channelQRCodeResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /channels/{id}/qr-code [get]
func (h *ChannelHandler) QRCode(c *gin.Context) {
	qrCode, err := h.runtimeService.GetQRCode(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, channelQRCodeResponse{
		ID:     c.Param("id"),
		QRCode: qrCode,
	})
}

func writeChannelError(c *gin.Context, err error) {
	var connectionErr *appservices.ChannelConnectionError
	switch {
	case errors.As(err, &connectionErr):
		_ = c.Error(response.New(400, connectionErr.Code, connectionErr.Message))
	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrChannelEncryptionFailed):
		_ = c.Error(response.New(400, channelErrorCode(err), channelErrorMessage(err)))
	case errors.Is(err, domain.ErrUnsupportedChannelKind),
		errors.Is(err, domain.ErrUnsupportedChannelState):
		_ = c.Error(response.Validation(validationDetailsForDomainError(err), channelErrorMessage(err)))
	case errors.Is(err, domain.ErrChannelAlreadyExists):
		_ = c.Error(response.New(409, "CHANNEL_ALREADY_EXISTS", "Channel name already exists."))
	case errors.Is(err, domain.ErrChannelNotFound):
		_ = c.Error(response.New(404, "CHANNEL_NOT_FOUND", "Channel not found."))
	default:
		_ = c.Error(response.InternalCause(err, ""))
	}
}

func validateCreateChannelRequest(req createChannelRequest) *response.APIError {
	return validateChannelPayload(
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.Kind),
		string(domain.ChannelStatusPending),
		req.Credentials,
		hasCredentialFields(req.Credentials),
	)
}

func validateUpdateChannelRequest(req updateChannelRequest) *response.APIError {
	credentials := req.Credentials
	requireCredentials := req.ReplaceCredentials
	if !requireCredentials {
		credentials = nil
	}
	return validateChannelPayload(
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.Kind),
		strings.TrimSpace(req.Status),
		credentials,
		requireCredentials,
	)
}

func validateTestChannelConnectionRequest(req testChannelConnectionRequest) *response.APIError {
	return validateChannelPayload(
		"test-connection",
		strings.TrimSpace(req.Kind),
		string(domain.ChannelStatusPending),
		req.Credentials,
		true,
	)
}

func validateChannelPayload(name, kind, status string, credentials json.RawMessage, requireCredentials bool) *response.APIError {
	details := map[string][]response.FieldError{}

	if strings.TrimSpace(name) == "" {
		addFieldError(details, "name", "REQUIRED", "Name is required.")
	}

	switch normalized := strings.TrimSpace(strings.ToLower(kind)); normalized {
	case string(domain.ChannelKindDiscord), string(domain.ChannelKindTelegram), string(domain.ChannelKindWhatsApp), string(domain.ChannelKindSlack), string(domain.ChannelKindWeb):
	default:
		addFieldError(details, "kind", "INVALID_VALUE", "Kind must be one of: telegram, discord, whatsapp, slack, web.")
	}

	switch normalized := strings.TrimSpace(strings.ToLower(status)); normalized {
	case string(domain.ChannelStatusPending), string(domain.ChannelStatusActive), string(domain.ChannelStatusDisabled):
	default:
		addFieldError(details, "status", "INVALID_VALUE", "Status must be one of: pending, active, disabled.")
	}

	if requireCredentials {
		addCredentialValidationErrors(details, strings.TrimSpace(strings.ToLower(kind)), credentials)
	}

	if len(details) == 0 {
		return nil
	}
	return response.Validation(details, "Channel validation failed")
}

func addCredentialValidationErrors(details map[string][]response.FieldError, kind string, credentials json.RawMessage) {
	switch kind {
	case string(domain.ChannelKindTelegram):
		var payload struct {
			Token string `json:"token"`
		}
		if !decodeCredentials(credentials, &payload) || strings.TrimSpace(payload.Token) == "" {
			addFieldError(details, "credentials.token", "REQUIRED", "Telegram bot token is required.")
		}
	case string(domain.ChannelKindDiscord):
		var payload struct {
			Token string `json:"token"`
		}
		if !decodeCredentials(credentials, &payload) || strings.TrimSpace(payload.Token) == "" {
			addFieldError(details, "credentials.token", "REQUIRED", "Discord bot token is required.")
		}
	case string(domain.ChannelKindSlack):
		var payload struct {
			BotToken string `json:"bot_token"`
			AppToken string `json:"app_token"`
		}
		if !decodeCredentials(credentials, &payload) || strings.TrimSpace(payload.BotToken) == "" {
			addFieldError(details, "credentials.bot_token", "REQUIRED", "Slack bot token is required.")
		}
		if !decodeCredentials(credentials, &payload) || strings.TrimSpace(payload.AppToken) == "" {
			addFieldError(details, "credentials.app_token", "REQUIRED", "Slack app token is required.")
		}
	}
}

func decodeCredentials(raw json.RawMessage, out any) bool {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return false
	}
	return json.Unmarshal(raw, out) == nil
}

func hasCredentialFields(raw json.RawMessage) bool {
	var payload map[string]any
	if !decodeCredentials(raw, &payload) {
		return false
	}
	return len(payload) > 0
}

func addFieldError(details map[string][]response.FieldError, field, code, message string) {
	details[field] = append(details[field], response.FieldError{
		Code:    code,
		Message: message,
	})
}

func channelErrorCode(err error) string {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return "CHANNEL_INVALID_INPUT"
	case errors.Is(err, domain.ErrUnsupportedChannelKind):
		return "CHANNEL_UNSUPPORTED_KIND"
	case errors.Is(err, domain.ErrUnsupportedChannelState):
		return "CHANNEL_UNSUPPORTED_STATUS"
	case errors.Is(err, domain.ErrChannelEncryptionFailed):
		return "CHANNEL_ENCRYPTION_FAILED"
	default:
		return "CHANNEL_BAD_REQUEST"
	}
}

func channelErrorMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return "Channel input is invalid."
	case errors.Is(err, domain.ErrUnsupportedChannelKind):
		return "Channel kind is not supported."
	case errors.Is(err, domain.ErrUnsupportedChannelState):
		return "Channel status is not supported."
	case errors.Is(err, domain.ErrChannelEncryptionFailed):
		return "Channel credentials could not be encrypted."
	default:
		return err.Error()
	}
}

func validationDetailsForDomainError(err error) map[string][]response.FieldError {
	details := map[string][]response.FieldError{}
	switch {
	case errors.Is(err, domain.ErrUnsupportedChannelKind):
		addFieldError(details, "kind", "INVALID_VALUE", "Kind must be one of: telegram, discord, whatsapp, slack, web.")
	case errors.Is(err, domain.ErrUnsupportedChannelState):
		addFieldError(details, "status", "INVALID_VALUE", "Status must be one of: pending, active, disabled.")
	}
	return details
}
