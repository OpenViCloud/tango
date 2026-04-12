package rest

import (
	"errors"
	"strings"

	"tango/internal/application/command"
	"tango/internal/application/query"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

type CloudflareConnectionHandler struct {
	create *command.CreateCloudflareConnectionHandler
	update *command.UpdateCloudflareConnectionHandler
	get    *query.GetCloudflareConnectionHandler
	list   *query.ListCloudflareConnectionsHandler
}

func NewCloudflareConnectionHandler(
	create *command.CreateCloudflareConnectionHandler,
	update *command.UpdateCloudflareConnectionHandler,
	get *query.GetCloudflareConnectionHandler,
	list *query.ListCloudflareConnectionsHandler,
) *CloudflareConnectionHandler {
	return &CloudflareConnectionHandler{
		create: create,
		update: update,
		get:    get,
		list:   list,
	}
}

func (h *CloudflareConnectionHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/cloudflare/connections", h.List)
	rg.GET("/cloudflare/connections/:id", h.Get)
	rg.POST("/cloudflare/connections", h.Create)
	rg.PUT("/cloudflare/connections/:id", h.Update)
}

type cloudflareConnectionRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	AccountID   string `json:"account_id" binding:"required"`
	ZoneID      string `json:"zone_id" binding:"required"`
	APIToken    string `json:"api_token"`
}

type cloudflareConnectionResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	AccountID   string `json:"account_id"`
	ZoneID      string `json:"zone_id"`
	Status      string `json:"status"`
	HasAPIToken bool   `json:"has_api_token"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (h *CloudflareConnectionHandler) List(c *gin.Context) {
	items, err := h.list.Handle(c.Request.Context(), query.ListCloudflareConnectionsQuery{
		UserID: c.GetString("user_id"),
	})
	if err != nil {
		writeCloudflareConnectionError(c, err)
		return
	}
	resp := make([]cloudflareConnectionResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCloudflareConnectionResponse(item))
	}
	response.OK(c, resp)
}

func (h *CloudflareConnectionHandler) Get(c *gin.Context) {
	item, err := h.get.Handle(c.Request.Context(), query.GetCloudflareConnectionQuery{
		UserID: c.GetString("user_id"),
		ID:     c.Param("id"),
	})
	if err != nil {
		writeCloudflareConnectionError(c, err)
		return
	}
	response.OK(c, toCloudflareConnectionResponse(item))
}

func (h *CloudflareConnectionHandler) Create(c *gin.Context) {
	var req cloudflareConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	if strings.TrimSpace(req.APIToken) == "" {
		_ = c.Error(response.BadRequest("api_token is required"))
		return
	}
	item, err := h.create.Handle(c.Request.Context(), command.CreateCloudflareConnectionCommand{
		UserID:      c.GetString("user_id"),
		DisplayName: req.DisplayName,
		AccountID:   req.AccountID,
		ZoneID:      req.ZoneID,
		APIToken:    req.APIToken,
	})
	if err != nil {
		writeCloudflareConnectionError(c, err)
		return
	}
	response.Created(c, toCloudflareConnectionResponse(item))
}

func (h *CloudflareConnectionHandler) Update(c *gin.Context) {
	var req cloudflareConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	item, err := h.update.Handle(c.Request.Context(), command.UpdateCloudflareConnectionCommand{
		UserID:      c.GetString("user_id"),
		ID:          c.Param("id"),
		DisplayName: req.DisplayName,
		AccountID:   req.AccountID,
		ZoneID:      req.ZoneID,
		APIToken:    req.APIToken,
	})
	if err != nil {
		writeCloudflareConnectionError(c, err)
		return
	}
	response.OK(c, toCloudflareConnectionResponse(item))
}

func toCloudflareConnectionResponse(item *domain.CloudflareConnection) cloudflareConnectionResponse {
	return cloudflareConnectionResponse{
		ID:          item.ID,
		DisplayName: item.DisplayName,
		AccountID:   item.AccountID,
		ZoneID:      item.ZoneID,
		Status:      string(item.Status),
		HasAPIToken: item.APITokenEncrypted != "",
		CreatedAt:   item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func writeCloudflareConnectionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrCloudflareConnectionNotFound):
		_ = c.Error(response.NotFound("cloudflare connection not found"))
	case errors.Is(err, domain.ErrInvalidInput):
		_ = c.Error(response.BadRequest("invalid cloudflare connection payload"))
	default:
		_ = c.Error(response.Internal("cloudflare connection request failed: " + err.Error()))
	}
}
