package rest

import (
	"errors"
	"time"

	"tango/internal/application/command"
	"tango/internal/application/query"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type APIKeyHandler struct {
	createKey *command.CreateAPIKeyHandler
	revokeKey *command.RevokeAPIKeyHandler
	listKeys  *query.ListAPIKeysHandler
}

func NewAPIKeyHandler(
	createKey *command.CreateAPIKeyHandler,
	revokeKey *command.RevokeAPIKeyHandler,
	listKeys *query.ListAPIKeysHandler,
) *APIKeyHandler {
	return &APIKeyHandler{
		createKey: createKey,
		revokeKey: revokeKey,
		listKeys:  listKeys,
	}
}

func (h *APIKeyHandler) Register(r *gin.RouterGroup) {
	r.POST("/api-keys", h.Create)
	r.GET("/api-keys", h.List)
	r.DELETE("/api-keys/:id", h.Revoke)
}

type createAPIKeyRequest struct {
	Name      string  `json:"name" binding:"required"`
	ExpiresAt *string `json:"expires_at"` // RFC3339, omit for no expiry
}

type apiKeyResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	UserID     string  `json:"user_id"`
	ExpiresAt  *string `json:"expires_at"`
	LastUsedAt *string `json:"last_used_at"`
	CreatedAt  string  `json:"created_at"`
}

type createAPIKeyResponse struct {
	apiKeyResponse
	Key string `json:"key"` // shown once only
}

// Create godoc
// @Summary Create API key
// @Description Creates a new API key for the authenticated user. The key is shown once and cannot be retrieved again.
// @Tags api-keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createAPIKeyRequest true "API key options"
// @Success 201 {object} createAPIKeyResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api-keys [post]
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req createAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			_ = c.Error(response.BadRequest("expires_at must be RFC3339 format"))
			return
		}
		utc := t.UTC()
		expiresAt = &utc
	}

	userID := c.GetString("user_id")
	result, err := h.createKey.Handle(c.Request.Context(), command.CreateAPIKeyCommand{
		ID:        uuid.New().String(),
		Name:      req.Name,
		UserID:    userID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAPIKeyNameEmpty):
			_ = c.Error(response.BadRequest(err.Error()))
		default:
			_ = c.Error(response.Internal(""))
		}
		return
	}

	response.Created(c, createAPIKeyResponse{
		apiKeyResponse: toAPIKeyResponse(result.APIKey),
		Key:            result.PlainKey,
	})
}

// List godoc
// @Summary List API keys
// @Description Lists all API keys belonging to the authenticated user. Key values are not returned.
// @Tags api-keys
// @Produce json
// @Security BearerAuth
// @Success 200 {array} apiKeyResponse
// @Failure 401 {object} map[string]string
// @Router /api-keys [get]
func (h *APIKeyHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	keys, err := h.listKeys.Handle(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(response.Internal(""))
		return
	}
	items := make([]apiKeyResponse, 0, len(keys))
	for _, k := range keys {
		items = append(items, toAPIKeyResponse(k))
	}
	response.OK(c, items)
}

// Revoke godoc
// @Summary Revoke API key
// @Description Permanently deletes an API key. Only the owner can revoke their own keys.
// @Tags api-keys
// @Produce json
// @Security BearerAuth
// @Param id path string true "API key ID"
// @Success 204
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api-keys/{id} [delete]
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	userID := c.GetString("user_id")
	if err := h.revokeKey.Handle(c.Request.Context(), command.RevokeAPIKeyCommand{
		ID:     c.Param("id"),
		UserID: userID,
	}); err != nil {
		switch {
		case errors.Is(err, domain.ErrAPIKeyNotFound):
			_ = c.Error(response.NotFound("api key not found"))
		default:
			_ = c.Error(response.Internal(""))
		}
		return
	}
	c.Status(204)
}

func toAPIKeyResponse(k *domain.APIKey) apiKeyResponse {
	r := apiKeyResponse{
		ID:        k.ID,
		Name:      k.Name,
		UserID:    k.UserID,
		CreatedAt: k.CreatedAt.Format(time.RFC3339),
	}
	if k.ExpiresAt != nil {
		s := k.ExpiresAt.Format(time.RFC3339)
		r.ExpiresAt = &s
	}
	if k.LastUsedAt != nil {
		s := k.LastUsedAt.Format(time.RFC3339)
		r.LastUsedAt = &s
	}
	return r
}
