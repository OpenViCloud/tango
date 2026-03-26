package rest

import (
	"errors"

	appservices "tango/internal/application/services"
	"tango/internal/contract/common"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	service appservices.RoleService
}

type createRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type roleResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type roleListResponse struct {
	Items      []roleResponse `json:"items"`
	PageIndex  int            `json:"pageIndex"`
	PageSize   int            `json:"pageSize"`
	TotalItems int64          `json:"totalItems"`
	TotalPage  int            `json:"totalPage"`
}

func NewRoleHandler(service appservices.RoleService) *RoleHandler {
	return &RoleHandler{service: service}
}

func (h *RoleHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/roles", h.List)
	rg.GET("/roles/:id", h.GetByID)
	rg.POST("/roles", h.Create)
	rg.PUT("/roles/:id", h.Update)
	rg.DELETE("/roles/:id", h.Delete)
}

// List godoc
// @Summary List roles
// @Tags roles
// @Produce json
// @Security BearerAuth
// @Success 200 {object} roleListResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context(), common.BaseRequestModel{
		PageIndex:  parseIntDefault(c.Query("pageIndex"), 0),
		PageSize:   parseIntDefault(c.Query("pageSize"), 20),
		SearchText: c.Query("searchText"),
		OrderBy:    c.Query("orderBy"),
		Ascending:  c.Query("ascending") == "true",
	})
	if err != nil {
		writeRoleError(c, err)
		return
	}
	response.OK(c, result)
}

// GetByID godoc
// @Summary Get role by ID
// @Tags roles
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Success 200 {object} roleResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /roles/{id} [get]
func (h *RoleHandler) GetByID(c *gin.Context) {
	view, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRoleError(c, err)
		return
	}
	response.OK(c, view)
}

// Create godoc
// @Summary Create role
// @Tags roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createRoleRequest true "Role payload"
// @Success 201 {object} roleResponse
// @Failure 400 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	view, err := h.service.Create(c.Request.Context(), appservices.CreateRoleInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeRoleError(c, err)
		return
	}
	response.Created(c, view)
}

// Update godoc
// @Summary Update role
// @Tags roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Param request body updateRoleRequest true "Role payload"
// @Success 200 {object} roleResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	view, err := h.service.Update(c.Request.Context(), appservices.UpdateRoleInput{
		ID:          c.Param("id"),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeRoleError(c, err)
		return
	}
	response.OK(c, view)
}

// Delete godoc
// @Summary Delete role
// @Tags roles
// @Produce json
// @Security BearerAuth
// @Param id path string true "Role ID"
// @Success 204
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeRoleError(c, err)
		return
	}
	response.NoContent(c)
}

func writeRoleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrRoleNotFound):
		_ = c.Error(response.NotFound(err.Error()))
	case errors.Is(err, domain.ErrRoleAlreadyExists):
		_ = c.Error(response.Conflict(err.Error()))
	case errors.Is(err, domain.ErrSystemRoleProtected),
		errors.Is(err, domain.ErrSystemRoleNameLocked),
		errors.Is(err, domain.ErrInvalidInput):
		_ = c.Error(response.BadRequest(err.Error()))
	default:
		_ = c.Error(response.InternalCause(err, ""))
	}
}
