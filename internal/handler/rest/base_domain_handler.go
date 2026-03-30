package rest

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"tango/internal/domain"
	response "tango/internal/handler/rest/response"
)

type BaseDomainHandler struct {
	baseDomainRepo     domain.BaseDomainRepository
	resourceDomainRepo domain.ResourceDomainRepository
	resourceRepo       domain.ResourceRepository
}

func NewBaseDomainHandler(
	baseDomainRepo domain.BaseDomainRepository,
	resourceDomainRepo domain.ResourceDomainRepository,
	resourceRepo domain.ResourceRepository,
) *BaseDomainHandler {
	return &BaseDomainHandler{
		baseDomainRepo:     baseDomainRepo,
		resourceDomainRepo: resourceDomainRepo,
		resourceRepo:       resourceRepo,
	}
}

func (h *BaseDomainHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/settings/base-domains", h.ListBaseDomains)
	rg.POST("/settings/base-domains", h.CreateBaseDomain)
	rg.DELETE("/settings/base-domains/:id", h.DeleteBaseDomain)
	rg.GET("/domains/check", h.CheckDomain)
}

type baseDomainResponse struct {
	ID              string    `json:"id"`
	Domain          string    `json:"domain"`
	WildcardEnabled bool      `json:"wildcard_enabled"`
	CreatedAt       time.Time `json:"created_at"`
}

func toBaseDomainResponse(bd *domain.BaseDomain) baseDomainResponse {
	return baseDomainResponse{
		ID:              bd.ID,
		Domain:          bd.Domain,
		WildcardEnabled: bd.WildcardEnabled,
		CreatedAt:       bd.CreatedAt,
	}
}

func (h *BaseDomainHandler) ListBaseDomains(c *gin.Context) {
	bds, err := h.baseDomainRepo.List(c.Request.Context())
	if err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	out := make([]baseDomainResponse, 0, len(bds))
	for _, bd := range bds {
		out = append(out, toBaseDomainResponse(bd))
	}
	response.OK(c, out)
}

func (h *BaseDomainHandler) CreateBaseDomain(c *gin.Context) {
	var req struct {
		Domain          string `json:"domain"           binding:"required"`
		WildcardEnabled bool   `json:"wildcard_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}

	ctx := c.Request.Context()

	// Check for duplicate
	if _, err := h.baseDomainRepo.GetByDomain(ctx, req.Domain); err == nil {
		_ = c.Error(response.Conflict(domain.ErrBaseDomainConflict.Error()))
		return
	}

	bd, err := h.baseDomainRepo.Create(ctx, domain.BaseDomain{
		ID:              uuid.NewString(),
		Domain:          req.Domain,
		WildcardEnabled: req.WildcardEnabled,
	})
	if err != nil {
		if errors.Is(err, domain.ErrBaseDomainConflict) {
			_ = c.Error(response.Conflict(err.Error()))
			return
		}
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	response.Created(c, toBaseDomainResponse(bd))
}

func (h *BaseDomainHandler) DeleteBaseDomain(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if _, err := h.baseDomainRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, domain.ErrBaseDomainNotFound) {
			_ = c.Error(response.NotFound("base domain not found"))
			return
		}
		_ = c.Error(response.InternalCause(err, ""))
		return
	}

	if err := h.baseDomainRepo.Delete(ctx, id); err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	response.NoContent(c)
}

func (h *BaseDomainHandler) CheckDomain(c *gin.Context) {
	host := c.Query("domain")
	if host == "" {
		_ = c.Error(response.Validation(nil, "domain query parameter is required"))
		return
	}

	ctx := c.Request.Context()

	rd, err := h.resourceDomainRepo.GetByHost(ctx, host)
	if err != nil {
		if errors.Is(err, domain.ErrResourceDomainNotFound) {
			c.JSON(http.StatusOK, gin.H{"available": true})
			return
		}
		_ = c.Error(response.InternalCause(err, ""))
		return
	}

	// Domain is in use — try to enrich with resource name
	res, resErr := h.resourceRepo.GetByID(ctx, rd.ResourceID)
	if resErr != nil {
		c.JSON(http.StatusOK, gin.H{
			"available":           false,
			"used_by_resource_id": rd.ResourceID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"available":             false,
		"used_by_resource_id":   rd.ResourceID,
		"used_by_resource_name": res.Name,
	})
}
