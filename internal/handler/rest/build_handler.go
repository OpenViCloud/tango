package rest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tango/internal/application/command"
	"tango/internal/application/query"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

type BuildHandler struct {
	create   *command.CreateBuildJobHandler
	upload   *command.CreateBuildJobFromUploadHandler
	cancel   *command.CancelBuildJobHandler
	getByID  *query.GetBuildJobHandler
	list     *query.ListBuildJobsHandler
}

func NewBuildHandler(
	create *command.CreateBuildJobHandler,
	upload *command.CreateBuildJobFromUploadHandler,
	cancel *command.CancelBuildJobHandler,
	getByID *query.GetBuildJobHandler,
	list *query.ListBuildJobsHandler,
) *BuildHandler {
	return &BuildHandler{
		create:  create,
		upload:  upload,
		cancel:  cancel,
		getByID: getByID,
		list:    list,
	}
}

func (h *BuildHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/builds", h.Create)
	rg.POST("/builds/upload", h.Upload)
	rg.GET("/builds/check-repo", h.CheckRepo)
	rg.GET("/builds", h.List)
	rg.GET("/builds/:id", h.Get)
	rg.POST("/builds/:id/cancel", h.Cancel)
}

// ── request / response DTOs ───────────────────────────────────────────────────

type createBuildRequest struct {
	GitURL    string `json:"git_url"    binding:"required"`
	GitBranch string `json:"git_branch"`
	BuildMode string `json:"build_mode"` // "auto" | "dockerfile"
	ImageTag  string `json:"image_tag"   binding:"required"`
}

type buildJobResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	SourceType  string `json:"source_type"`
	BuildMode   string `json:"build_mode"`
	GitURL      string `json:"git_url"`
	GitBranch   string `json:"git_branch"`
	ArchiveName string `json:"archive_name,omitempty"`
	ImageTag    string `json:"image_tag"`
	Logs        string `json:"logs"`
	ErrorMsg    string `json:"error_msg,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	FinishedAt  string `json:"finished_at,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
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
// @Summary Submit a new build job from a Git URL
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
		BuildMode: req.BuildMode,
		ImageTag:  req.ImageTag,
	})
	if err != nil {
		writeBuildError(c, err)
		return
	}
	c.JSON(202, toBuildJobResponse(job))
}

// Upload godoc
// @Summary Submit a new build job from an uploaded archive (.tar.gz or .zip)
// @Tags builds
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param archive formData file true "Source archive (.tar.gz, .tgz, or .zip)"
// @Param image_tag formData string true "Image tag to push"
// @Param build_mode formData string false "Build mode: auto (default) or dockerfile"
// @Success 202 {object} buildJobResponse
// @Router /builds/upload [post]
func (h *BuildHandler) Upload(c *gin.Context) {
	imageTag := strings.TrimSpace(c.PostForm("image_tag"))
	if imageTag == "" {
		_ = c.Error(response.BadRequest("image_tag is required"))
		return
	}
	buildMode := strings.TrimSpace(c.PostForm("build_mode"))

	fileHeader, err := c.FormFile("archive")
	if err != nil {
		_ = c.Error(response.BadRequest("archive file is required"))
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	// Handle double extension .tar.gz
	if strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".tar.gz") {
		ext = ".tar.gz"
	}
	if ext != ".tar.gz" && ext != ".tgz" && ext != ".zip" {
		_ = c.Error(response.BadRequest("only .tar.gz, .tgz, or .zip archives are supported"))
		return
	}

	destPath := filepath.Join(os.TempDir(), "tango-upload-"+newUploadID()+ext)
	if err := c.SaveUploadedFile(fileHeader, destPath); err != nil {
		_ = c.Error(response.InternalCause(err, "save upload"))
		return
	}

	job, err := h.upload.Handle(c.Request.Context(), command.CreateBuildJobFromUploadCommand{
		ArchivePath: destPath,
		ArchiveName: fileHeader.Filename,
		BuildMode:   buildMode,
		ImageTag:    imageTag,
	})
	if err != nil {
		_ = os.Remove(destPath)
		_ = c.Error(response.InternalCause(err, ""))
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

// CheckRepo godoc
// @Summary Check if a git repository is accessible and optionally verify a branch
// @Tags builds
// @Produce json
// @Security BearerAuth
// @Param url    query string true  "Git repository URL"
// @Param branch query string false "Branch to verify (optional)"
// @Param token  query string false "Access token for private repos (optional)"
// @Success 200 {object} checkRepoResponse
// @Router /builds/check-repo [get]
func (h *BuildHandler) CheckRepo(c *gin.Context) {
	rawURL := strings.TrimSpace(c.Query("url"))
	if rawURL == "" {
		_ = c.Error(response.BadRequest("url is required"))
		return
	}
	branch := strings.TrimSpace(c.Query("branch"))
	token := strings.TrimSpace(c.Query("token"))

	gitURL := rawURL
	if token != "" {
		const httpsPrefix = "https://"
		if strings.HasPrefix(rawURL, httpsPrefix) {
			gitURL = httpsPrefix + token + "@" + rawURL[len(httpsPrefix):]
		}
	}

	checkCtx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// Check reachability + detect default branch via symref
	symrefCmd := exec.CommandContext(checkCtx, "git", "ls-remote", "--symref", gitURL, "HEAD")
	symrefOut, symrefErr := symrefCmd.CombinedOutput()
	if symrefErr != nil {
		response.OK(c, checkRepoResponse{
			Available: false,
			Error:     "repository not accessible or does not exist",
		})
		return
	}

	resp := checkRepoResponse{Available: true}
	for _, line := range strings.Split(string(symrefOut), "\n") {
		if strings.HasPrefix(line, "ref: refs/heads/") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				resp.DefaultBranch = strings.TrimPrefix(parts[0], "ref: refs/heads/")
			}
			break
		}
	}

	if branch != "" {
		branchCmd := exec.CommandContext(checkCtx, "git", "ls-remote", gitURL, fmt.Sprintf("refs/heads/%s", branch))
		branchOut, branchErr := branchCmd.Output()
		resp.BranchExists = branchErr == nil && len(strings.TrimSpace(string(branchOut))) > 0
	}

	response.OK(c, resp)
}

type checkRepoResponse struct {
	Available     bool   `json:"available"`
	DefaultBranch string `json:"default_branch,omitempty"`
	BranchExists  bool   `json:"branch_exists"`
	Error         string `json:"error,omitempty"`
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toBuildJobResponse(j *domain.BuildJob) buildJobResponse {
	r := buildJobResponse{
		ID:          j.ID,
		Status:      string(j.Status),
		SourceType:  j.SourceType,
		BuildMode:   j.BuildMode,
		GitURL:      j.GitURL,
		GitBranch:   j.GitBranch,
		ArchiveName: j.ArchiveName,
		ImageTag:    j.ImageTag,
		Logs:        j.Logs,
		ErrorMsg:    j.ErrorMsg,
		CreatedAt:   j.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   j.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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

func newUploadID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
