package rest

import (
	"errors"
	"fmt"
	"time"

	"github.com/docker/docker/errdefs"
	"github.com/gin-gonic/gin"

	"tango/internal/application/command"
	"tango/internal/application/query"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"
)

// DockerHandler exposes Docker container and image management endpoints.
type DockerHandler struct {
	listContainers    *query.ListContainersHandler
	listImages        *query.ListImagesHandler
	getContainer      *query.GetContainerDetailsHandler
	getContainerStats *query.GetContainerStatsHandler
	createContainer   *command.CreateContainerHandler
	startContainer    *command.StartContainerHandler
	stopContainer     *command.StopContainerHandler
	removeContainer   *command.RemoveContainerHandler
	pullImage         *command.PullImageHandler
	removeImage       *command.RemoveImageHandler
}

func NewDockerHandler(
	listContainers *query.ListContainersHandler,
	listImages *query.ListImagesHandler,
	getContainer *query.GetContainerDetailsHandler,
	getContainerStats *query.GetContainerStatsHandler,
	createContainer *command.CreateContainerHandler,
	startContainer *command.StartContainerHandler,
	stopContainer *command.StopContainerHandler,
	removeContainer *command.RemoveContainerHandler,
	pullImage *command.PullImageHandler,
	removeImage *command.RemoveImageHandler,
) *DockerHandler {
	return &DockerHandler{
		listContainers:    listContainers,
		listImages:        listImages,
		getContainer:      getContainer,
		getContainerStats: getContainerStats,
		createContainer:   createContainer,
		startContainer:    startContainer,
		stopContainer:     stopContainer,
		removeContainer:   removeContainer,
		pullImage:         pullImage,
		removeImage:       removeImage,
	}
}

func (h *DockerHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/docker/containers", h.ListContainers)
	rg.GET("/docker/containers/:id", h.GetContainer)
	rg.GET("/docker/containers/:id/stats", h.GetContainerStats)
	rg.POST("/docker/containers", h.CreateContainer)
	rg.POST("/docker/containers/:id/start", h.StartContainer)
	rg.POST("/docker/containers/:id/stop", h.StopContainer)
	rg.DELETE("/docker/containers/:id", h.RemoveContainer)

	rg.GET("/docker/images", h.ListImages)
	rg.POST("/docker/images/pull", h.PullImage)
	rg.DELETE("/docker/images/:id", h.RemoveImage)
}

// ── Request / Response types ──────────────────────────────────────────────────

type containerPortResponse struct {
	IP          string `json:"ip"`
	PrivatePort uint16 `json:"private_port"`
	PublicPort  uint16 `json:"public_port"`
	Type        string `json:"type"`
}

type containerResponse struct {
	ID      string                  `json:"id"`
	ShortID string                  `json:"short_id"`
	Name    string                  `json:"name"`
	Image   string                  `json:"image"`
	ImageID string                  `json:"image_id"`
	State   string                  `json:"state"`
	Status  string                  `json:"status"`
	Command string                  `json:"command"`
	Ports   []containerPortResponse `json:"ports"`
	Labels  map[string]string       `json:"labels"`
}

type containerMountResponse struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Driver      string `json:"driver"`
	Mode        string `json:"mode"`
	RW          bool   `json:"rw"`
}

type containerDetailsResponse struct {
	ID           string                   `json:"id"`
	ShortID      string                   `json:"short_id"`
	Name         string                   `json:"name"`
	Image        string                   `json:"image"`
	ImageID      string                   `json:"image_id"`
	Command      []string                 `json:"command"`
	CreatedAt    string                   `json:"created_at"`
	State        string                   `json:"state"`
	Status       string                   `json:"status"`
	StartedAt    string                   `json:"started_at"`
	FinishedAt   string                   `json:"finished_at"`
	ExitCode     int                      `json:"exit_code"`
	Error        string                   `json:"error"`
	RestartCount int                      `json:"restart_count"`
	Ports        []containerPortResponse  `json:"ports"`
	Labels       map[string]string        `json:"labels"`
	Networks     map[string]string        `json:"networks"`
	Mounts       []containerMountResponse `json:"mounts"`
}

type containerStatsResponse struct {
	ReadAt           string  `json:"read_at"`
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryUsageBytes uint64  `json:"memory_usage_bytes"`
	MemoryLimitBytes uint64  `json:"memory_limit_bytes"`
	MemoryPercent    float64 `json:"memory_percent"`
	NetworkRxBytes   uint64  `json:"network_rx_bytes"`
	NetworkTxBytes   uint64  `json:"network_tx_bytes"`
	BlockReadBytes   uint64  `json:"block_read_bytes"`
	BlockWriteBytes  uint64  `json:"block_write_bytes"`
	PidsCurrent      uint64  `json:"pids_current"`
}

type imageResponse struct {
	ID        string   `json:"id"`
	ShortID   string   `json:"short_id"`
	Tags      []string `json:"tags"`
	Size      string   `json:"size"`
	SizeBytes int64    `json:"size_bytes"`
	Created   string   `json:"created"`
	Digest    string   `json:"digest"`
	InUse     int64    `json:"in_use"`
}

type createContainerRequest struct {
	Name         string            `json:"name"`
	Image        string            `json:"image"`
	Cmd          []string          `json:"cmd"`
	Env          map[string]string `json:"env"`
	PortBindings map[string]string `json:"port_bindings"` // containerPort -> hostPort, e.g. "80" -> "8080"
	Volumes      []string          `json:"volumes"`       // bind mounts, e.g. "/host:/container"
	AutoRemove   bool              `json:"auto_remove"`
}

type pullImageRequest struct {
	Reference string `json:"reference"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// ListContainers godoc
// @Summary List containers
// @Tags docker
// @Produce json
// @Security BearerAuth
// @Param all query bool false "Include stopped containers"
// @Success 200 {array} containerResponse
// @Failure 500 {object} errorResponse
// @Router /docker/containers [get]
func (h *DockerHandler) ListContainers(c *gin.Context) {
	all := c.Query("all") == "true"
	containers, err := h.listContainers.Handle(c.Request.Context(), query.ListContainersQuery{All: all})
	if err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	items := make([]containerResponse, 0, len(containers))
	for _, ct := range containers {
		items = append(items, toContainerResponse(ct))
	}
	response.OK(c, items)
}

func (h *DockerHandler) GetContainer(c *gin.Context) {
	ct, err := h.getContainer.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(writeDockerError(err))
		return
	}
	response.OK(c, toContainerDetailsResponse(ct))
}

func (h *DockerHandler) GetContainerStats(c *gin.Context) {
	stats, err := h.getContainerStats.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(writeDockerError(err))
		return
	}
	response.OK(c, toContainerStatsResponse(stats))
}

// CreateContainer godoc
// @Summary Create a container
// @Tags docker
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createContainerRequest true "Container config"
// @Success 201 {object} containerResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /docker/containers [post]
func (h *DockerHandler) CreateContainer(c *gin.Context) {
	var req createContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	if req.Image == "" {
		_ = c.Error(response.Validation(nil, "image is required"))
		return
	}

	ct, err := h.createContainer.Handle(c.Request.Context(), command.CreateContainerCommand{
		Name:         req.Name,
		Image:        req.Image,
		Cmd:          req.Cmd,
		Env:          req.Env,
		PortBindings: req.PortBindings,
		Volumes:      req.Volumes,
		AutoRemove:   req.AutoRemove,
	})
	if err != nil {
		if errdefs.IsInvalidParameter(err) || errdefs.IsNotFound(err) || errdefs.IsConflict(err) {
			_ = c.Error(response.BadRequest(err.Error()))
		} else {
			_ = c.Error(response.InternalCause(err, ""))
		}
		return
	}
	response.Created(c, toContainerResponse(ct))
}

// StartContainer godoc
// @Summary Start a container
// @Tags docker
// @Produce json
// @Security BearerAuth
// @Param id path string true "Container ID"
// @Success 204
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /docker/containers/{id}/start [post]
func (h *DockerHandler) StartContainer(c *gin.Context) {
	if err := h.startContainer.Handle(c.Request.Context(), command.StartContainerCommand{
		ContainerID: c.Param("id"),
	}); err != nil {
		_ = c.Error(writeDockerError(err))
		return
	}
	response.NoContent(c)
}

// StopContainer godoc
// @Summary Stop a container
// @Tags docker
// @Produce json
// @Security BearerAuth
// @Param id path string true "Container ID"
// @Success 204
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /docker/containers/{id}/stop [post]
func (h *DockerHandler) StopContainer(c *gin.Context) {
	if err := h.stopContainer.Handle(c.Request.Context(), command.StopContainerCommand{
		ContainerID: c.Param("id"),
	}); err != nil {
		_ = c.Error(writeDockerError(err))
		return
	}
	response.NoContent(c)
}

// RemoveContainer godoc
// @Summary Remove a container
// @Tags docker
// @Produce json
// @Security BearerAuth
// @Param id path string true "Container ID"
// @Param force query bool false "Force remove running container"
// @Success 204
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /docker/containers/{id} [delete]
func (h *DockerHandler) RemoveContainer(c *gin.Context) {
	force := c.Query("force") == "true"
	if err := h.removeContainer.Handle(c.Request.Context(), command.RemoveContainerCommand{
		ContainerID: c.Param("id"),
		Force:       force,
	}); err != nil {
		_ = c.Error(writeDockerError(err))
		return
	}
	response.NoContent(c)
}

// ListImages godoc
// @Summary List images
// @Tags docker
// @Produce json
// @Security BearerAuth
// @Success 200 {array} imageResponse
// @Failure 500 {object} errorResponse
// @Router /docker/images [get]
func (h *DockerHandler) ListImages(c *gin.Context) {
	images, err := h.listImages.Handle(c.Request.Context())
	if err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	items := make([]imageResponse, 0, len(images))
	for _, img := range images {
		items = append(items, toImageResponse(img))
	}
	response.OK(c, items)
}

// PullImage godoc
// @Summary Pull an image from a registry
// @Tags docker
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body pullImageRequest true "Image reference"
// @Success 204
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /docker/images/pull [post]
func (h *DockerHandler) PullImage(c *gin.Context) {
	var req pullImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	if req.Reference == "" {
		_ = c.Error(response.Validation(nil, "reference is required"))
		return
	}
	if err := h.pullImage.Handle(c.Request.Context(), command.PullImageCommand{
		Reference: req.Reference,
	}); err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	response.NoContent(c)
}

// RemoveImage godoc
// @Summary Remove an image
// @Tags docker
// @Produce json
// @Security BearerAuth
// @Param id path string true "Image ID or tag"
// @Param force query bool false "Force remove"
// @Success 204
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /docker/images/{id} [delete]
func (h *DockerHandler) RemoveImage(c *gin.Context) {
	force := c.Query("force") == "true"
	if err := h.removeImage.Handle(c.Request.Context(), command.RemoveImageCommand{
		ImageID: c.Param("id"),
		Force:   force,
	}); err != nil {
		_ = c.Error(writeDockerError(err))
		return
	}
	response.NoContent(c)
}

// ── Mappers ───────────────────────────────────────────────────────────────────

func toContainerResponse(ct domain.Container) containerResponse {
	ports := make([]containerPortResponse, 0, len(ct.Ports))
	for _, p := range ct.Ports {
		ports = append(ports, containerPortResponse{
			IP:          p.IP,
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Type:        p.Type,
		})
	}
	return containerResponse{
		ID:      ct.ID,
		ShortID: shortID(ct.ID),
		Name:    ct.Name,
		Image:   ct.Image,
		ImageID: ct.ImageID,
		State:   ct.State,
		Status:  ct.Status,
		Command: ct.Command,
		Ports:   ports,
		Labels:  ct.Labels,
	}
}

func toImageResponse(img domain.Image) imageResponse {
	created := ""
	if img.Created > 0 {
		created = time.Unix(img.Created, 0).UTC().Format(time.RFC3339)
	}
	return imageResponse{
		ID:        img.ID,
		ShortID:   shortID(img.ID),
		Tags:      img.Tags,
		Size:      formatBytes(img.Size),
		SizeBytes: img.Size,
		Created:   created,
		Digest:    img.Digest,
		InUse:     img.InUse,
	}
}

func toContainerDetailsResponse(ct domain.ContainerDetails) containerDetailsResponse {
	ports := make([]containerPortResponse, 0, len(ct.Ports))
	for _, p := range ct.Ports {
		ports = append(ports, containerPortResponse{
			IP:          p.IP,
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Type:        p.Type,
		})
	}

	mounts := make([]containerMountResponse, 0, len(ct.Mounts))
	for _, mount := range ct.Mounts {
		mounts = append(mounts, containerMountResponse{
			Type:        mount.Type,
			Name:        mount.Name,
			Source:      mount.Source,
			Destination: mount.Destination,
			Driver:      mount.Driver,
			Mode:        mount.Mode,
			RW:          mount.RW,
		})
	}

	return containerDetailsResponse{
		ID:           ct.ID,
		ShortID:      shortID(ct.ID),
		Name:         ct.Name,
		Image:        ct.Image,
		ImageID:      ct.ImageID,
		Command:      ct.Command,
		CreatedAt:    ct.CreatedAt,
		State:        ct.State,
		Status:       ct.Status,
		StartedAt:    ct.StartedAt,
		FinishedAt:   ct.FinishedAt,
		ExitCode:     ct.ExitCode,
		Error:        ct.Error,
		RestartCount: ct.RestartCount,
		Ports:        ports,
		Labels:       ct.Labels,
		Networks:     ct.Networks,
		Mounts:       mounts,
	}
}

func toContainerStatsResponse(stats domain.ContainerStats) containerStatsResponse {
	return containerStatsResponse{
		ReadAt:           stats.ReadAt,
		CPUPercent:       stats.CPUPercent,
		MemoryUsageBytes: stats.MemoryUsageBytes,
		MemoryLimitBytes: stats.MemoryLimitBytes,
		MemoryPercent:    stats.MemoryPercent,
		NetworkRxBytes:   stats.NetworkRxBytes,
		NetworkTxBytes:   stats.NetworkTxBytes,
		BlockReadBytes:   stats.BlockReadBytes,
		BlockWriteBytes:  stats.BlockWriteBytes,
		PidsCurrent:      stats.PidsCurrent,
	}
}

func shortID(id string) string {
	// Docker IDs are prefixed with "sha256:"
	id = trimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func writeDockerError(err error) *response.APIError {
	if errors.Is(err, domain.ErrContainerNotFound) || errors.Is(err, domain.ErrImageNotFound) {
		return response.New(404, "DOCKER_NOT_FOUND", err.Error())
	}
	return response.InternalCause(err, "")
}
