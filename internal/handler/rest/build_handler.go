package rest

import (
	"errors"
	"strconv"

	"tango/internal/application/command"
	"tango/internal/application/query"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

type BuildHandler struct {
	create   *command.CreateBuildJobHandler
	cancel   *command.CancelBuildJobHandler
	getByID  *query.GetBuildJobHandler
	list     *query.ListBuildJobsHandler
}

func NewBuildHandler(
	create *command.CreateBuildJobHandler,
	cancel *command.CancelBuildJobHandler,
	getByID *query.GetBuildJobHandler,
	list *query.ListBuildJobsHandler,
) *BuildHandler {
	return &BuildHandler{
		create:  create,
		cancel:  cancel,
		getByID: getByID,
		list:    list,
	}
}

func (h *BuildHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/builds", h.Create)
	rg.GET("/builds", h.List)
	rg.GET("/builds/:id", h.Get)
	rg.POST("/builds/:id/cancel", h.Cancel)
}

// ── request / response DTOs ───────────────────────────────────────────────────

type createBuildRequest struct {
	GitURL    string `json:"git_url"   binding:"required"`
	GitBranch string `json:"git_branch"`
	ImageTag  string `json:"image_tag"  binding:"required"`
}

type buildJobResponse struct {
	ID         string  `json:"id"`
	Status     string  `json:"status"`
	GitURL     string  `json:"git_url"`
	GitBranch  string  `json:"git_branch"`
	ImageTag   string  `json:"image_tag"`
	Logs       string  `json:"logs"`
	ErrorMsg   string  `json:"error_msg,omitempty"`
	StartedAt  string  `json:"started_at,omitempty"`
	FinishedAt string  `json:"finished_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type buildJobListResponse struct {
	Items      []buildJobResponse `json:"items"`
	TotalItems int64              `json:"total_items"`
	TotalPages int                `json:"total_pages"`
	PageIndex  int                `json:"page_index"`
	PageSize   int                `json:"page_size"`
}

// ── handlers ─────────────────────────────────────────────────────────────────

// Create godoc
// @Summary Submit a new build job
// @Tags builds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createBuildRequest true "Build request"
// @Success 202 {object} buildJobResponse
// @Router /builds [post]
func (h *BuildHandler) Create(c *gin.Context) {
	var req createBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}
	job, err := h.create.Handle(c.Request.Context(), command.CreateBuildJobCommand{
		GitURL:    req.GitURL,
		GitBranch: req.GitBranch,
		ImageTag:  req.ImageTag,
	})
	if err != nil {
		writeBuildError(c, err)
		return
	}
	c.JSON(202, toBuildJobResponse(job))
}

// Get godoc
// @Summary Get a build job by ID
// @Tags builds
// @Produce json
// @Security BearerAuth
// @Param id path string true "Build job ID"
// @Success 200 {object} buildJobResponse
// @Router /builds/{id} [get]
func (h *BuildHandler) Get(c *gin.Context) {
	job, err := h.getByID.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeBuildError(c, err)
		return
	}
	response.OK(c, toBuildJobResponse(job))
}

// List godoc
// @Summary List build jobs
// @Tags builds
// @Produce json
// @Security BearerAuth
// @Param pageIndex query int false "Page index (0-based)"
// @Param pageSize  query int false "Page size"
// @Param status    query string false "Filter by status"
// @Success 200 {object} buildJobListResponse
// @Router /builds [get]
func (h *BuildHandler) List(c *gin.Context) {
	pageIndex, _ := strconv.Atoi(c.DefaultQuery("pageIndex", "0"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status")

	result, err := h.list.Handle(c.Request.Context(), query.ListBuildJobsQuery{
		PageIndex: pageIndex,
		PageSize:  pageSize,
		Status:    status,
	})
	if err != nil {
		writeBuildError(c, err)
		return
	}

	totalPages := 0
	if result.PageSize > 0 {
		totalPages = int((result.TotalItems + int64(result.PageSize) - 1) / int64(result.PageSize))
	}
	items := make([]buildJobResponse, 0, len(result.Items))
	for _, j := range result.Items {
		items = append(items, toBuildJobResponse(j))
	}
	response.OK(c, buildJobListResponse{
		Items:      items,
		TotalItems: result.TotalItems,
		TotalPages: totalPages,
		PageIndex:  result.PageIndex,
		PageSize:   result.PageSize,
	})
}

// Cancel godoc
// @Summary Cancel a running build job
// @Tags builds
// @Produce json
// @Security BearerAuth
// @Param id path string true "Build job ID"
// @Success 200 {object} buildJobResponse
// @Router /builds/{id}/cancel [post]
func (h *BuildHandler) Cancel(c *gin.Context) {
	job, err := h.cancel.Handle(c.Request.Context(), command.CancelBuildJobCommand{
		ID: c.Param("id"),
	})
	if err != nil {
		writeBuildError(c, err)
		return
	}
	response.OK(c, toBuildJobResponse(job))
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toBuildJobResponse(j *domain.BuildJob) buildJobResponse {
	r := buildJobResponse{
		ID:        j.ID,
		Status:    string(j.Status),
		GitURL:    j.GitURL,
		GitBranch: j.GitBranch,
		ImageTag:  j.ImageTag,
		Logs:      j.Logs,
		ErrorMsg:  j.ErrorMsg,
		CreatedAt: j.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: j.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if j.StartedAt != nil {
		r.StartedAt = j.StartedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if j.FinishedAt != nil {
		r.FinishedAt = j.FinishedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return r
}

func writeBuildError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrBuildJobNotFound):
		_ = c.Error(response.NotFound(err.Error()))
	case errors.Is(err, domain.ErrBuildJobNotCancelable):
		_ = c.Error(response.BadRequest(err.Error()))
	default:
		_ = c.Error(response.InternalCause(err, ""))
	}
}
