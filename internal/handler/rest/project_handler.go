package rest

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"tango/internal/application/command"
	"tango/internal/application/query"
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
	createResource        *command.CreateResourceHandler
	createResourceFromGit *command.CreateResourceFromGitHandler
	updateResource        *command.UpdateResourceHandler
	startResourceRun      *command.CreateStartResourceRunHandler
	stopResource          *command.StopResourceHandler
	deleteResource        *command.DeleteResourceHandler
	setEnvVars            *command.SetResourceEnvVarsHandler
	listProjects          *query.ListProjectsHandler
	getProject            *query.GetProjectHandler
	listEnvResources      *query.ListEnvironmentResourcesHandler
	getResource           *query.GetResourceHandler
	dockerRepo            domain.DockerRepository
}

func NewProjectHandler(
	createProject *command.CreateProjectHandler,
	updateProject *command.UpdateProjectHandler,
	deleteProject *command.DeleteProjectHandler,
	createEnvironment *command.CreateEnvironmentHandler,
	deleteEnvironment *command.DeleteEnvironmentHandler,
	createResource *command.CreateResourceHandler,
	createResourceFromGit *command.CreateResourceFromGitHandler,
	updateResource *command.UpdateResourceHandler,
	startResourceRun *command.CreateStartResourceRunHandler,
	stopResource *command.StopResourceHandler,
	deleteResource *command.DeleteResourceHandler,
	setEnvVars *command.SetResourceEnvVarsHandler,
	listProjects *query.ListProjectsHandler,
	getProject *query.GetProjectHandler,
	listEnvResources *query.ListEnvironmentResourcesHandler,
	getResource *query.GetResourceHandler,
	dockerRepo domain.DockerRepository,
) *ProjectHandler {
	return &ProjectHandler{
		createProject:         createProject,
		updateProject:         updateProject,
		deleteProject:         deleteProject,
		createEnvironment:     createEnvironment,
		deleteEnvironment:     deleteEnvironment,
		createResource:        createResource,
		createResourceFromGit: createResourceFromGit,
		updateResource:        updateResource,
		startResourceRun:      startResourceRun,
		stopResource:          stopResource,
		deleteResource:        deleteResource,
		setEnvVars:            setEnvVars,
		listProjects:          listProjects,
		getProject:            getProject,
		listEnvResources:      listEnvResources,
		getResource:           getResource,
		dockerRepo:            dockerRepo,
	}
}

func (h *ProjectHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/projects", h.ListProjects)
	rg.POST("/projects", h.CreateProject)
	rg.GET("/projects/:id", h.GetProject)
	rg.PUT("/projects/:id", h.UpdateProject)
	rg.DELETE("/projects/:id", h.DeleteProject)
	rg.POST("/projects/:id/environments", h.CreateEnvironment)
	rg.DELETE("/environments/:envId", h.DeleteEnvironment)
	rg.GET("/environments/:envId/resources", h.ListResources)
	rg.POST("/environments/:envId/resources", h.CreateResource)
	rg.POST("/environments/:envId/resources/from-git", h.CreateResourceFromGit)
	rg.GET("/resources/:resourceId", h.GetResource)
	rg.PUT("/resources/:resourceId", h.UpdateResource)
	rg.DELETE("/resources/:resourceId", h.DeleteResource)
	rg.POST("/resources/:resourceId/start", h.StartResource)
	rg.POST("/resources/:resourceId/stop", h.StopResource)
	rg.GET("/resources/:resourceId/logs", h.GetResourceLogs)
	rg.GET("/resources/:resourceId/env-vars", h.GetEnvVars)
	rg.PUT("/resources/:resourceId/env-vars", h.SetEnvVars)
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
	Ports   []createResourcePortRequest   `json:"ports"`
	EnvVars []createResourceEnvVarRequest `json:"env_vars"`
}

type createResourceFromGitRequest struct {
	Name      string                        `json:"name"       binding:"required"`
	GitURL    string                        `json:"git_url"    binding:"required"`
	GitBranch string                        `json:"git_branch"`
	BuildMode string                        `json:"build_mode"` // "auto" | "dockerfile"
	GitToken  string                        `json:"git_token"`
	ImageTag  string                        `json:"image_tag"  binding:"required"`
	Ports     []createResourcePortRequest   `json:"ports"`
	EnvVars   []createResourceEnvVarRequest `json:"env_vars"`
}

type updateResourceRequest struct {
	Name  string                        `json:"name"`
	Ports []createResourcePortRequest   `json:"ports"`
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
	SourceType    string         `json:"source_type,omitempty"`
	GitURL        string         `json:"git_url,omitempty"`
	BuildJobID    string         `json:"build_job_id,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	Ports         []portResponse `json:"ports"`
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
		ID:    c.Param("resourceId"),
		Name:  req.Name,
		Ports: ports,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	response.OK(c, toResourceResponse(resource))
}

func (h *ProjectHandler) DeleteResource(c *gin.Context) {
	if err := h.deleteResource.Handle(c.Request.Context(), command.DeleteResourceCommand{ID: c.Param("resourceId")}); err != nil {
		writeProjectError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *ProjectHandler) StartResource(c *gin.Context) {
	run, err := h.startResourceRun.Handle(c.Request.Context(), command.CreateStartResourceRunCommand{
		ResourceID: c.Param("resourceId"),
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(202, response.SuccessEnvelope[resourceRunResponse]{
		Success: true,
		TraceID: response.TraceID(c),
		Data:    toResourceRunResponse(run),
	})
}

func (h *ProjectHandler) StopResource(c *gin.Context) {
	if err := h.stopResource.Handle(c.Request.Context(), command.StopResourceCommand{ID: c.Param("resourceId")}); err != nil {
		writeProjectError(c, err)
		return
	}
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
		SourceType:    r.SourceType,
		GitURL:        r.GitURL,
		BuildJobID:    r.BuildJobID,
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

func writeProjectError(c *gin.Context, err error) {
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
	default:
		_ = c.Error(response.InternalCause(err, ""))
	}
}
