package rest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"tango/internal/application/command"
	"tango/internal/application/query"
	appservices "tango/internal/application/services"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"
)

// ProjectHandler exposes project, environment, and resource management endpoints.
type ProjectHandler struct {
	createProject         *command.CreateProjectHandler
	updateProject         *command.UpdateProjectHandler
	deleteProject         *command.DeleteProjectHandler
	createEnvironment     *command.CreateEnvironmentHandler
	deleteEnvironment     *command.DeleteEnvironmentHandler
	forkEnvironment       *command.ForkEnvironmentHandler
	createResource        *command.CreateResourceHandler
	createResourceFromGit *command.CreateResourceFromGitHandler
	startBuildForResource *command.StartBuildForResourceHandler
	updateResource        *command.UpdateResourceHandler
	startResourceRun      *command.CreateStartResourceRunHandler
	stopResource          *command.StopResourceHandler
	deleteResource        *command.DeleteResourceHandler
	setEnvVars            *command.SetResourceEnvVarsHandler
	listProjects          *query.ListProjectsHandler
	getProject            *query.GetProjectHandler
	listResourceTemplates *query.ListResourceTemplatesHandler
	listEnvResources      *query.ListEnvironmentResourcesHandler
	getResource           *query.GetResourceHandler
	runtimeReconciler     appservices.ResourceRuntimeReconciler
	dockerRepo            domain.DockerRepository
	domainRepo            domain.ResourceDomainRepository
	platformConfigRepo    domain.PlatformConfigRepository
	fileProvider          domain.TraefikFileProvider
	cache                 appservices.Cache
}

func NewProjectHandler(
	createProject *command.CreateProjectHandler,
	updateProject *command.UpdateProjectHandler,
	deleteProject *command.DeleteProjectHandler,
	createEnvironment *command.CreateEnvironmentHandler,
	deleteEnvironment *command.DeleteEnvironmentHandler,
	forkEnvironment *command.ForkEnvironmentHandler,
	createResource *command.CreateResourceHandler,
	createResourceFromGit *command.CreateResourceFromGitHandler,
	startBuildForResource *command.StartBuildForResourceHandler,
	updateResource *command.UpdateResourceHandler,
	startResourceRun *command.CreateStartResourceRunHandler,
	stopResource *command.StopResourceHandler,
	deleteResource *command.DeleteResourceHandler,
	setEnvVars *command.SetResourceEnvVarsHandler,
	listProjects *query.ListProjectsHandler,
	getProject *query.GetProjectHandler,
	listResourceTemplates *query.ListResourceTemplatesHandler,
	listEnvResources *query.ListEnvironmentResourcesHandler,
	getResource *query.GetResourceHandler,
	runtimeReconciler appservices.ResourceRuntimeReconciler,
	dockerRepo domain.DockerRepository,
	domainRepo domain.ResourceDomainRepository,
	platformConfigRepo domain.PlatformConfigRepository,
	fileProvider domain.TraefikFileProvider,
	cache appservices.Cache,
) *ProjectHandler {
	return &ProjectHandler{
		createProject:         createProject,
		updateProject:         updateProject,
		deleteProject:         deleteProject,
		createEnvironment:     createEnvironment,
		deleteEnvironment:     deleteEnvironment,
		forkEnvironment:       forkEnvironment,
		createResource:        createResource,
		createResourceFromGit: createResourceFromGit,
		startBuildForResource: startBuildForResource,
		updateResource:        updateResource,
		startResourceRun:      startResourceRun,
		stopResource:          stopResource,
		deleteResource:        deleteResource,
		setEnvVars:            setEnvVars,
		listProjects:          listProjects,
		getProject:            getProject,
		listResourceTemplates: listResourceTemplates,
		listEnvResources:      listEnvResources,
		getResource:           getResource,
		runtimeReconciler:     runtimeReconciler,
		dockerRepo:            dockerRepo,
		domainRepo:            domainRepo,
		platformConfigRepo:    platformConfigRepo,
		fileProvider:          fileProvider,
		cache:                 cache,
	}
}

func (h *ProjectHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/projects", h.ListProjects)
	rg.POST("/projects", h.CreateProject)
	rg.GET("/projects/:id", h.GetProject)
	rg.PUT("/projects/:id", h.UpdateProject)
	rg.DELETE("/projects/:id", h.DeleteProject)
	rg.GET("/resource-templates", h.ListResourceTemplates)
	rg.POST("/projects/:id/environments", h.CreateEnvironment)
	rg.POST("/environments/:envId/fork", h.ForkEnvironment)
	rg.DELETE("/environments/:envId", h.DeleteEnvironment)
	rg.GET("/environments/:envId/resources", h.ListResources)
	rg.POST("/environments/:envId/resources", h.CreateResource)
	rg.POST("/environments/:envId/resources/from-git", h.CreateResourceFromGit)
	rg.POST("/resources/:resourceId/build", h.StartBuildForResource)
	rg.GET("/resources/:resourceId", h.GetResource)
	rg.POST("/resources/reconcile", h.ReconcileResources)
	rg.POST("/resources/:resourceId/reconcile", h.ReconcileResource)
	rg.GET("/resources/:resourceId/connection-info", h.GetResourceConnectionInfo)
	rg.PUT("/resources/:resourceId", h.UpdateResource)
	rg.DELETE("/resources/:resourceId", h.DeleteResource)
	rg.POST("/resources/:resourceId/start", h.StartResource)
	rg.POST("/resources/:resourceId/stop", h.StopResource)
	rg.GET("/resources/:resourceId/logs", h.GetResourceLogs)
	rg.GET("/resources/:resourceId/env-vars", h.GetEnvVars)
	rg.PUT("/resources/:resourceId/env-vars", h.SetEnvVars)
	rg.GET("/resources/:resourceId/domains", h.ListResourceDomains)
	rg.POST("/resources/:resourceId/domains", h.AddResourceDomain)
	rg.PATCH("/resources/:resourceId/domains/:domainId", h.UpdateResourceDomain)
	rg.DELETE("/resources/:resourceId/domains/:domainId", h.RemoveResourceDomain)
	rg.POST("/resources/:resourceId/domains/:domainId/verify", h.VerifyResourceDomain)
}

// ── Request / Response types ──────────────────────────────────────────────────

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createEnvironmentRequest struct {
	Name string `json:"name"`
}

type forkEnvironmentRequest struct {
	Name string `json:"name" binding:"required"`
}

type createResourcePortRequest struct {
	HostPort     int    `json:"host_port"`
	InternalPort int    `json:"internal_port"`
	Proto        string `json:"proto"`
	Label        string `json:"label"`
}

type createResourceEnvVarRequest struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
}

type createResourceRequest struct {
	Name    string                        `json:"name"`
	Type    string                        `json:"type"`
	Image   string                        `json:"image"`
	Tag     string                        `json:"tag"`
	Config  map[string]any                `json:"config"`
	NodeID  *string                       `json:"node_id"`
	Ports   []createResourcePortRequest   `json:"ports"`
	EnvVars []createResourceEnvVarRequest `json:"env_vars"`
}

type createResourceFromGitRequest struct {
	Name         string                        `json:"name"          binding:"required"`
	ConnectionID string                        `json:"connection_id"`
	GitURL       string                        `json:"git_url"       binding:"required"`
	GitBranch    string                        `json:"git_branch"`
	BuildMode    string                        `json:"build_mode"` // "auto" | "dockerfile"
	GitToken     string                        `json:"git_token"`
	ImageTag     string                        `json:"image_tag"     binding:"required"`
	Ports        []createResourcePortRequest   `json:"ports"`
	EnvVars      []createResourceEnvVarRequest `json:"env_vars"`
}

type updateResourceRequest struct {
	Name   string                      `json:"name"`
	Ports  []createResourcePortRequest `json:"ports"`
	Config map[string]any              `json:"config"`
}

type setEnvVarsRequest struct {
	Vars []createResourceEnvVarRequest `json:"vars"`
}

type portResponse struct {
	ID           string `json:"id"`
	HostPort     int    `json:"host_port"`
	InternalPort int    `json:"internal_port"`
	Proto        string `json:"proto"`
	Label        string `json:"label"`
}

type envVarResponse struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	IsSecret bool   `json:"is_secret"`
}

type resourceResponse struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	Status        string         `json:"status"`
	Image         string         `json:"image"`
	Tag           string         `json:"tag"`
	ContainerID   string         `json:"container_id"`
	Config        map[string]any `json:"config"`
	EnvironmentID string         `json:"environment_id"`
	NodeID        *string        `json:"node_id,omitempty"`
	SourceType    string         `json:"source_type,omitempty"`
	GitURL        string         `json:"git_url,omitempty"`
	BuildJobID    string         `json:"build_job_id,omitempty"`
	ImageTag      string         `json:"image_tag,omitempty"`
	ConnectionID  string         `json:"connection_id,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	Ports         []portResponse `json:"ports"`
}

type resourceTemplatePortResponse struct {
	Host      string `json:"host"`
	Container string `json:"container"`
}

type resourceTemplateEnvVarResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type resourceTemplateResponse struct {
	ID          string                           `json:"id"`
	Name        string                           `json:"name"`
	IconURL     string                           `json:"icon_url"`
	Image       string                           `json:"image"`
	Description string                           `json:"description"`
	Color       string                           `json:"color"`
	Abbr        string                           `json:"abbr"`
	Tags        []string                         `json:"tags"`
	Ports       []resourceTemplatePortResponse   `json:"ports"`
	Env         []resourceTemplateEnvVarResponse `json:"env"`
	Type        string                           `json:"type"`
	Volumes     []string                         `json:"volumes,omitempty"`
	Cmd         []string                         `json:"cmd,omitempty"`
}

type resourceRunResponse struct {
	ID         string `json:"id"`
	ResourceID string `json:"resource_id"`
	Status     string `json:"status"`
	Logs       string `json:"logs"`
	ErrorMsg   string `json:"error_msg,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type resourceLogsResponse struct {
	ResourceID  string   `json:"resource_id"`
	ContainerID string   `json:"container_id"`
	Status      string   `json:"status"`
	Lines       []string `json:"lines"`
}

type resourceConnectionPortResponse struct {
	ID               string `json:"id"`
	HostPort         int    `json:"host_port"`
	InternalPort     int    `json:"internal_port"`
	Label            string `json:"label"`
	InternalEndpoint string `json:"internal_endpoint"`
	ExternalEndpoint string `json:"external_endpoint,omitempty"`
}

type resourceConnectionInfoResponse struct {
	ResourceID   string                           `json:"resource_id"`
	InternalHost string                           `json:"internal_host"`
	ExternalHost string                           `json:"external_host,omitempty"`
	Ports        []resourceConnectionPortResponse `json:"ports"`
}

type envResponse struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	ProjectID string             `json:"project_id"`
	CreatedAt string             `json:"created_at"`
	Resources []resourceResponse `json:"resources"`
}

type projectResponse struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	CreatedBy    string        `json:"created_by"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
	Environments []envResponse `json:"environments"`
}

type resourceRuntimeReconcileResponse struct {
	Checked           int `json:"checked"`
	Updated           int `json:"updated"`
	Running           int `json:"running"`
	Stopped           int `json:"stopped"`
	Errored           int `json:"errored"`
	MissingContainers int `json:"missing_containers"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	projects, err := h.listProjects.Handle(c.Request.Context(), query.ListProjectsQuery{})
	if err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	resp := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		resp = append(resp, toProjectResponse(p))
	}
	response.OK(c, resp)
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	userID := c.GetString("user_id")
	project, err := h.createProject.Handle(c.Request.Context(), command.CreateProjectCommand{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Created(c, toProjectResponse(project))
}

func (h *ProjectHandler) GetProject(c *gin.Context) {
	project, err := h.getProject.Handle(c.Request.Context(), query.GetProjectQuery{ID: c.Param("id")})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.OK(c, toProjectResponse(project))
}

func (h *ProjectHandler) ReconcileResources(c *gin.Context) {
	if h.runtimeReconciler == nil {
		_ = c.Error(response.BadRequest("resource runtime reconciler is unavailable"))
		return
	}

	summary, err := h.runtimeReconciler.ReconcileAll(c.Request.Context())
	if err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	response.OK(c, toResourceRuntimeReconcileResponse(summary))
}

func (h *ProjectHandler) ReconcileResource(c *gin.Context) {
	if h.runtimeReconciler == nil {
		_ = c.Error(response.BadRequest("resource runtime reconciler is unavailable"))
		return
	}

	summary, err := h.runtimeReconciler.ReconcileResource(c.Request.Context(), c.Param("resourceId"))
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.OK(c, toResourceRuntimeReconcileResponse(summary))
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	var req updateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	project, err := h.updateProject.Handle(c.Request.Context(), command.UpdateProjectCommand{
		ID:          c.Param("id"),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.OK(c, toProjectResponse(project))
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	if err := h.deleteProject.Handle(c.Request.Context(), command.DeleteProjectCommand{ID: c.Param("id")}); err != nil {
		writeProjectError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *ProjectHandler) ListResourceTemplates(c *gin.Context) {
	templates, err := h.listResourceTemplates.Handle(c.Request.Context(), query.ListResourceTemplatesQuery{})
	if err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}

	resp := make([]resourceTemplateResponse, 0, len(templates))
	for _, template := range templates {
		ports := make([]resourceTemplatePortResponse, 0, len(template.Ports))
		for _, port := range template.Ports {
			ports = append(ports, resourceTemplatePortResponse{
				Host:      port.Host,
				Container: port.Container,
			})
		}

		env := make([]resourceTemplateEnvVarResponse, 0, len(template.Env))
		for _, entry := range template.Env {
			env = append(env, resourceTemplateEnvVarResponse{
				Key:   entry.Key,
				Value: entry.Value,
			})
		}

		resp = append(resp, resourceTemplateResponse{
			ID:          template.ID,
			Name:        template.Name,
			IconURL:     template.IconURL,
			Image:       template.Image,
			Description: template.Description,
			Color:       template.Color,
			Abbr:        template.Abbr,
			Tags:        template.Tags,
			Ports:       ports,
			Env:         env,
			Type:        template.Type,
			Volumes:     template.Volumes,
			Cmd:         template.Cmd,
		})
	}

	response.OK(c, resp)
}

func (h *ProjectHandler) CreateEnvironment(c *gin.Context) {
	var req createEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	env, err := h.createEnvironment.Handle(c.Request.Context(), command.CreateEnvironmentCommand{
		ID:        uuid.New().String(),
		Name:      req.Name,
		ProjectID: c.Param("id"),
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Created(c, toEnvResponse(env))
}

func (h *ProjectHandler) DeleteEnvironment(c *gin.Context) {
	if err := h.deleteEnvironment.Handle(c.Request.Context(), command.DeleteEnvironmentCommand{ID: c.Param("envId")}); err != nil {
		writeProjectError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *ProjectHandler) ForkEnvironment(c *gin.Context) {
	var req forkEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	userID := c.GetString("user_id")
	env, err := h.forkEnvironment.Handle(c.Request.Context(), command.ForkEnvironmentCommand{
		SourceEnvID: c.Param("envId"),
		NewEnvID:    uuid.New().String(),
		Name:        req.Name,
		CreatedBy:   userID,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Created(c, toEnvResponse(env))
}

func (h *ProjectHandler) ListResources(c *gin.Context) {
	resources, err := h.listEnvResources.Handle(c.Request.Context(), query.ListEnvironmentResourcesQuery{
		EnvironmentID: c.Param("envId"),
	})
	if err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	resp := make([]resourceResponse, 0, len(resources))
	for _, r := range resources {
		resp = append(resp, toResourceResponse(r))
	}
	response.OK(c, resp)
}

func (h *ProjectHandler) CreateResource(c *gin.Context) {
	var req createResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	userID := c.GetString("user_id")

	ports := make([]command.ResourcePortInput, 0, len(req.Ports))
	for _, p := range req.Ports {
		ports = append(ports, command.ResourcePortInput{
			HostPort:     p.HostPort,
			InternalPort: p.InternalPort,
			Proto:        p.Proto,
			Label:        p.Label,
		})
	}

	envVars := make([]command.ResourceEnvVarInput, 0, len(req.EnvVars))
	for _, ev := range req.EnvVars {
		envVars = append(envVars, command.ResourceEnvVarInput{
			Key:      ev.Key,
			Value:    ev.Value,
			IsSecret: ev.IsSecret,
		})
	}

	resource, err := h.createResource.Handle(c.Request.Context(), command.CreateResourceCommand{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Type:          req.Type,
		Image:         req.Image,
		Tag:           req.Tag,
		Config:        req.Config,
		NodeID:        req.NodeID,
		EnvironmentID: c.Param("envId"),
		CreatedBy:     userID,
		Ports:         ports,
		EnvVars:       envVars,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Created(c, toResourceResponse(resource))
}

func (h *ProjectHandler) CreateResourceFromGit(c *gin.Context) {
	var req createResourceFromGitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	userID := c.GetString("user_id")

	ports := make([]command.ResourcePortInput, 0, len(req.Ports))
	for _, p := range req.Ports {
		ports = append(ports, command.ResourcePortInput{
			HostPort:     p.HostPort,
			InternalPort: p.InternalPort,
			Proto:        p.Proto,
			Label:        p.Label,
		})
	}
	envVars := make([]command.ResourceEnvVarInput, 0, len(req.EnvVars))
	for _, ev := range req.EnvVars {
		envVars = append(envVars, command.ResourceEnvVarInput{
			Key:      ev.Key,
			Value:    ev.Value,
			IsSecret: ev.IsSecret,
		})
	}

	resource, err := h.createResourceFromGit.Handle(c.Request.Context(), command.CreateResourceFromGitCommand{
		Name:          req.Name,
		EnvironmentID: c.Param("envId"),
		CreatedBy:     userID,
		ConnectionID:  req.ConnectionID,
		GitURL:        req.GitURL,
		GitBranch:     req.GitBranch,
		BuildMode:     req.BuildMode,
		GitToken:      req.GitToken,
		ImageTag:      req.ImageTag,
		Ports:         ports,
		EnvVars:       envVars,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.Created(c, toResourceResponse(resource))
}

func (h *ProjectHandler) GetResource(c *gin.Context) {
	resource, err := h.getResource.Handle(c.Request.Context(), query.GetResourceQuery{ID: c.Param("resourceId")})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.OK(c, toResourceResponse(resource))
}

func (h *ProjectHandler) GetResourceConnectionInfo(c *gin.Context) {
	info, err := h.getResourceConnectionInfo(c.Request.Context(), c.Param("resourceId"))
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.OK(c, info)
}

func (h *ProjectHandler) UpdateResource(c *gin.Context) {
	var req updateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	ports := make([]command.ResourcePortInput, 0, len(req.Ports))
	for _, p := range req.Ports {
		ports = append(ports, command.ResourcePortInput{
			HostPort:     p.HostPort,
			InternalPort: p.InternalPort,
			Proto:        p.Proto,
			Label:        p.Label,
		})
	}
	resource, err := h.updateResource.Handle(c.Request.Context(), command.UpdateResourceCommand{
		ID:     c.Param("resourceId"),
		Name:   req.Name,
		Ports:  ports,
		Config: req.Config,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	h.invalidateResourceConnectionCache(c.Request.Context(), resource.ID)
	response.OK(c, toResourceResponse(resource))
}

func (h *ProjectHandler) DeleteResource(c *gin.Context) {
	resourceID := c.Param("resourceId")
	if err := h.deleteResource.Handle(c.Request.Context(), command.DeleteResourceCommand{ID: resourceID}); err != nil {
		writeProjectError(c, err)
		return
	}
	h.invalidateResourceConnectionCache(c.Request.Context(), resourceID)
	response.NoContent(c)
}

func (h *ProjectHandler) StartBuildForResource(c *gin.Context) {
	userID := c.GetString("user_id")
	job, err := h.startBuildForResource.Handle(c.Request.Context(), command.StartBuildForResourceCommand{
		ResourceID: c.Param("resourceId"),
		UserID:     userID,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.OK(c, gin.H{"build_job_id": job.ID})
}

func (h *ProjectHandler) StartResource(c *gin.Context) {
	run, err := h.startResourceRun.Handle(c.Request.Context(), command.CreateStartResourceRunCommand{
		ResourceID: c.Param("resourceId"),
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	h.invalidateResourceConnectionCache(c.Request.Context(), c.Param("resourceId"))
	c.JSON(202, response.SuccessEnvelope[resourceRunResponse]{
		Success: true,
		TraceID: response.TraceID(c),
		Data:    toResourceRunResponse(run),
	})
}

func (h *ProjectHandler) StopResource(c *gin.Context) {
	resourceID := c.Param("resourceId")
	if err := h.stopResource.Handle(c.Request.Context(), command.StopResourceCommand{ID: resourceID}); err != nil {
		writeProjectError(c, err)
		return
	}
	h.invalidateResourceConnectionCache(c.Request.Context(), resourceID)
	response.NoContent(c)
}

func (h *ProjectHandler) GetResourceLogs(c *gin.Context) {
	resource, err := h.getResource.Handle(c.Request.Context(), query.GetResourceQuery{ID: c.Param("resourceId")})
	if err != nil {
		writeProjectError(c, err)
		return
	}

	resp := resourceLogsResponse{
		ResourceID:  resource.ID,
		ContainerID: resource.ContainerID,
		Status:      resource.Status,
		Lines:       []string{},
	}

	if resource.ContainerID == "" {
		response.OK(c, resp)
		return
	}
	if h.dockerRepo == nil {
		response.OK(c, resp)
		return
	}

	tail := c.DefaultQuery("tail", "200")
	lines, err := h.dockerRepo.GetContainerLogs(c.Request.Context(), resource.ContainerID, domain.GetContainerLogsInput{
		Tail: tail,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	resp.Lines = lines
	response.OK(c, resp)
}

func (h *ProjectHandler) GetEnvVars(c *gin.Context) {
	resource, err := h.getResource.Handle(c.Request.Context(), query.GetResourceQuery{ID: c.Param("resourceId")})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	resp := make([]envVarResponse, 0, len(resource.EnvVars))
	for _, ev := range resource.EnvVars {
		resp = append(resp, toEnvVarResponse(ev))
	}
	response.OK(c, resp)
}

func (h *ProjectHandler) SetEnvVars(c *gin.Context) {
	var req setEnvVarsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	vars := make([]command.ResourceEnvVarInput, 0, len(req.Vars))
	for _, v := range req.Vars {
		vars = append(vars, command.ResourceEnvVarInput{
			Key:      v.Key,
			Value:    v.Value,
			IsSecret: v.IsSecret,
		})
	}
	if err := h.setEnvVars.Handle(c.Request.Context(), command.SetResourceEnvVarsCommand{
		ResourceID: c.Param("resourceId"),
		Vars:       vars,
	}); err != nil {
		writeProjectError(c, err)
		return
	}
	response.NoContent(c)
}

// ── Mappers ───────────────────────────────────────────────────────────────────

func toProjectResponse(p *domain.Project) projectResponse {
	envs := make([]envResponse, 0, len(p.Environments))
	for i := range p.Environments {
		envs = append(envs, toEnvResponse(&p.Environments[i]))
	}
	return projectResponse{
		ID:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		CreatedBy:    p.CreatedBy,
		CreatedAt:    p.CreatedAt.Format(timeLayout),
		UpdatedAt:    p.UpdatedAt.Format(timeLayout),
		Environments: envs,
	}
}

func toEnvResponse(e *domain.Environment) envResponse {
	resources := make([]resourceResponse, 0, len(e.Resources))
	for i := range e.Resources {
		resources = append(resources, toResourceResponse(&e.Resources[i]))
	}
	return envResponse{
		ID:        e.ID,
		Name:      e.Name,
		ProjectID: e.ProjectID,
		CreatedAt: e.CreatedAt.Format(timeLayout),
		Resources: resources,
	}
}

func toResourceResponse(r *domain.Resource) resourceResponse {
	ports := make([]portResponse, 0, len(r.Ports))
	for _, p := range r.Ports {
		ports = append(ports, portResponse{
			ID:           p.ID,
			HostPort:     p.HostPort,
			InternalPort: p.InternalPort,
			Proto:        p.Proto,
			Label:        p.Label,
		})
	}
	return resourceResponse{
		ID:            r.ID,
		Name:          r.Name,
		Type:          r.Type,
		Status:        r.Status,
		Image:         r.Image,
		Tag:           r.Tag,
		ContainerID:   r.ContainerID,
		Config:        r.Config,
		EnvironmentID: r.EnvironmentID,
		NodeID:        r.NodeID,
		SourceType:    r.SourceType,
		GitURL:        r.GitURL,
		BuildJobID:    r.BuildJobID,
		ImageTag:      r.ImageTag,
		ConnectionID:  r.ConnectionID,
		CreatedAt:     r.CreatedAt.Format(timeLayout),
		UpdatedAt:     r.UpdatedAt.Format(timeLayout),
		Ports:         ports,
	}
}

func toEnvVarResponse(ev domain.ResourceEnvVar) envVarResponse {
	resp := envVarResponse{
		ID:       ev.ID,
		Key:      ev.Key,
		IsSecret: ev.IsSecret,
	}
	if !ev.IsSecret {
		resp.Value = ev.Value
	}
	return resp
}

func toResourceRunResponse(run *domain.ResourceRun) resourceRunResponse {
	resp := resourceRunResponse{
		ID:         run.ID,
		ResourceID: run.ResourceID,
		Status:     string(run.Status),
		Logs:       run.Logs,
		ErrorMsg:   run.ErrorMsg,
		CreatedAt:  run.CreatedAt.Format(timeLayout),
		UpdatedAt:  run.UpdatedAt.Format(timeLayout),
	}
	if run.StartedAt != nil {
		resp.StartedAt = run.StartedAt.Format(timeLayout)
	}
	if run.FinishedAt != nil {
		resp.FinishedAt = run.FinishedAt.Format(timeLayout)
	}
	return resp
}

func toResourceRuntimeReconcileResponse(summary *appservices.ResourceRuntimeReconcileSummary) resourceRuntimeReconcileResponse {
	if summary == nil {
		return resourceRuntimeReconcileResponse{}
	}
	return resourceRuntimeReconcileResponse{
		Checked:           summary.Checked,
		Updated:           summary.Updated,
		Running:           summary.Running,
		Stopped:           summary.Stopped,
		Errored:           summary.Errored,
		MissingContainers: summary.MissingContainers,
	}
}

func writeProjectError(c *gin.Context, err error) {
	var portConflict *domain.ErrHostPortConflict
	switch {
	case errors.Is(err, domain.ErrProjectNotFound):
		_ = c.Error(response.NotFound(err.Error()))
	case errors.Is(err, domain.ErrEnvironmentNotFound):
		_ = c.Error(response.NotFound(err.Error()))
	case errors.Is(err, domain.ErrResourceNotFound):
		_ = c.Error(response.NotFound(err.Error()))
	case errors.Is(err, domain.ErrResourceRunNotFound):
		_ = c.Error(response.NotFound(err.Error()))
	case errors.Is(err, domain.ErrResourceNotStarted):
		_ = c.Error(response.BadRequest(err.Error()))
	case errors.As(err, &portConflict):
		_ = c.Error(response.BadRequest(portConflict.Error()))
	case domain.IsUserFacing(err):
		var ufErr *domain.UserFacingError
		errors.As(err, &ufErr)
		_ = c.Error(response.BadRequest(ufErr.Error()))
	default:
		_ = c.Error(response.InternalCause(err, ""))
	}
}

func (h *ProjectHandler) getResourceConnectionInfo(ctx context.Context, resourceID string) (resourceConnectionInfoResponse, error) {
	return appservices.GetOrCreate(ctx, h.cache, resourceConnectionCacheKey(resourceID), 2*time.Minute, func(ctx context.Context) (resourceConnectionInfoResponse, error) {
		resource, err := h.getResource.Handle(ctx, query.GetResourceQuery{ID: resourceID})
		if err != nil {
			return resourceConnectionInfoResponse{}, err
		}

		internalHost := strings.TrimSpace(resource.Name)
		if h.dockerRepo != nil && strings.TrimSpace(resource.ContainerID) != "" {
			if info, err := h.dockerRepo.InspectContainer(ctx, resource.ContainerID); err == nil && strings.TrimSpace(info.Name) != "" {
				internalHost = info.Name
			}
		}

		externalHost := ""
		if h.platformConfigRepo != nil {
			if cfg, err := h.platformConfigRepo.Get(ctx, domain.PlatformConfigPublicIP); err == nil {
				externalHost = strings.TrimSpace(cfg.Value)
			}
		}

		ports := make([]resourceConnectionPortResponse, 0, len(resource.Ports))
		for _, p := range resource.Ports {
			portInfo := resourceConnectionPortResponse{
				ID:               p.ID,
				HostPort:         p.HostPort,
				InternalPort:     p.InternalPort,
				Label:            p.Label,
				InternalEndpoint: fmt.Sprintf("%s:%d", internalHost, p.InternalPort),
			}
			if externalHost != "" && p.HostPort > 0 {
				portInfo.ExternalEndpoint = fmt.Sprintf("%s:%d", externalHost, p.HostPort)
			}
			ports = append(ports, portInfo)
		}

		return resourceConnectionInfoResponse{
			ResourceID:   resource.ID,
			InternalHost: internalHost,
			ExternalHost: externalHost,
			Ports:        ports,
		}, nil
	})
}

func (h *ProjectHandler) invalidateResourceConnectionCache(ctx context.Context, resourceID string) {
	if h.cache == nil || strings.TrimSpace(resourceID) == "" {
		return
	}
	_ = h.cache.Delete(ctx, resourceConnectionCacheKey(resourceID))
}

func resourceConnectionCacheKey(resourceID string) string {
	return "resource_connection_info:" + resourceID
}

// ── Resource Domain handlers ──────────────────────────────────────────────────

type resourceDomainResponse struct {
	ID         string     `json:"id"`
	ResourceID string     `json:"resource_id"`
	Host       string     `json:"host"`
	TargetPort int        `json:"target_port"`
	Type       string     `json:"type"`
	TLSEnabled bool       `json:"tls_enabled"`
	Verified   bool       `json:"verified"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func toResourceDomainResponse(d *domain.ResourceDomain) resourceDomainResponse {
	return resourceDomainResponse{
		ID:         d.ID,
		ResourceID: d.ResourceID,
		Host:       d.Host,
		TargetPort: d.TargetPort,
		Type:       d.Type,
		TLSEnabled: d.TLSEnabled,
		Verified:   d.Verified,
		VerifiedAt: d.VerifiedAt,
		CreatedAt:  d.CreatedAt,
	}
}

func (h *ProjectHandler) ListResourceDomains(c *gin.Context) {
	resourceID := c.Param("resourceId")
	domains, err := h.domainRepo.ListByResource(c.Request.Context(), resourceID)
	if err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	out := make([]resourceDomainResponse, 0, len(domains))
	for _, d := range domains {
		out = append(out, toResourceDomainResponse(d))
	}
	c.JSON(http.StatusOK, out)
}

func (h *ProjectHandler) AddResourceDomain(c *gin.Context) {
	resourceID := c.Param("resourceId")
	var req struct {
		Host       string `json:"host" binding:"required"`
		TargetPort int    `json:"target_port" binding:"required"`
		TLSEnabled bool   `json:"tls_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	d, err := h.domainRepo.Create(c.Request.Context(), domain.ResourceDomain{
		ID:         uuid.NewString(),
		ResourceID: resourceID,
		Host:       req.Host,
		TargetPort: req.TargetPort,
		TLSEnabled: req.TLSEnabled,
		Type:       domain.ResourceDomainTypeCustom,
		Verified:   false,
	})
	if err != nil {
		if errors.Is(err, domain.ErrResourceDomainConflict) {
			_ = c.Error(response.BadRequest("domain already in use"))
			return
		}
		_ = c.Error(response.InternalCause(err, ""))
		return
	}
	c.JSON(http.StatusCreated, toResourceDomainResponse(d))
}

func (h *ProjectHandler) UpdateResourceDomain(c *gin.Context) {
	ctx := c.Request.Context()
	domainID := c.Param("domainId")

	var req struct {
		Host       string `json:"host" binding:"required"`
		TargetPort int    `json:"target_port" binding:"required"`
		TLSEnabled bool   `json:"tls_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	current, err := h.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		_ = c.Error(response.NotFound("domain not found"))
		return
	}

	hostChanged := current.Host != req.Host
	updated := *current
	updated.Host = req.Host
	updated.TargetPort = req.TargetPort
	updated.TLSEnabled = req.TLSEnabled
	if hostChanged {
		updated.Verified = false
		updated.VerifiedAt = nil
	}

	next, err := h.domainRepo.Update(ctx, updated)
	if err != nil {
		if errors.Is(err, domain.ErrResourceDomainConflict) {
			_ = c.Error(response.BadRequest("domain already in use"))
			return
		}
		_ = c.Error(response.InternalCause(err, ""))
		return
	}

	h.refreshTraefikConfig(ctx, next.ResourceID)
	c.JSON(http.StatusOK, toResourceDomainResponse(next))
}

func (h *ProjectHandler) RemoveResourceDomain(c *gin.Context) {
	ctx := c.Request.Context()
	domainID := c.Param("domainId")

	d, err := h.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		_ = c.Error(response.NotFound("domain not found"))
		return
	}

	if err := h.domainRepo.Delete(ctx, domainID); err != nil {
		_ = c.Error(response.InternalCause(err, ""))
		return
	}

	// Refresh Traefik config immediately (best-effort)
	h.refreshTraefikConfig(ctx, d.ResourceID)

	c.Status(http.StatusNoContent)
}

func (h *ProjectHandler) VerifyResourceDomain(c *gin.Context) {
	domainID := c.Param("domainId")
	ctx := c.Request.Context()

	d, err := h.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		_ = c.Error(response.NotFound("domain not found"))
		return
	}

	// Resolve DNS and compare with configured public IP
	addrs, err := net.LookupHost(d.Host)
	if err != nil || len(addrs) == 0 {
		c.JSON(http.StatusOK, gin.H{"verified": false, "reason": "DNS lookup failed"})
		return
	}

	publicIP := ""
	if h.platformConfigRepo != nil {
		if cfg, cfgErr := h.platformConfigRepo.Get(ctx, domain.PlatformConfigPublicIP); cfgErr == nil {
			publicIP = cfg.Value
		}
	}

	verified := false
	for _, addr := range addrs {
		if addr == publicIP {
			verified = true
			break
		}
	}

	if verified {
		_ = h.domainRepo.SetVerified(ctx, d.ID, time.Now())
		// Refresh Traefik config immediately now that domain is verified (best-effort)
		h.refreshTraefikConfig(ctx, d.ResourceID)
	}

	c.JSON(http.StatusOK, gin.H{"verified": verified, "resolved_ips": addrs})
}

// refreshTraefikConfig rewrites the Traefik file config for a resource using its
// current container (if running) and latest domain list.
func (h *ProjectHandler) refreshTraefikConfig(ctx context.Context, resourceID string) {
	if h.fileProvider == nil || h.dockerRepo == nil {
		return
	}

	resource, err := h.getResource.Handle(ctx, query.GetResourceQuery{ID: resourceID})
	if err != nil || resource.ContainerID == "" {
		return
	}

	info, err := h.dockerRepo.InspectContainer(ctx, resource.ContainerID)
	if err != nil {
		return
	}

	domains, err := h.domainRepo.ListByResource(ctx, resourceID)
	if err != nil || len(domains) == 0 {
		_ = h.fileProvider.Delete(resourceID)
		return
	}

	certResolver := ""
	if h.platformConfigRepo != nil {
		if cfg, err := h.platformConfigRepo.Get(ctx, domain.PlatformConfigCertResolver); err == nil {
			certResolver = cfg.Value
		}
	}

	_ = h.fileProvider.Write(resourceID, domains, info.Name, certResolver)
}
