package command

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"tango/internal/domain"
)

// ── ResourceAutoStarter ───────────────────────────────────────────────────────
// Implemented by ResourceRunService; called by BuildService after a successful
// build when the job has a ResourceID attached.

type ResourceAutoStarter interface {
	StartAfterBuild(ctx context.Context, resourceID, imageTag string) error
}

// ── CreateResourceFromGit ─────────────────────────────────────────────────────
// Saves the resource record to the database WITHOUT starting a build.
// The user triggers a build later from the resource detail page.

type CreateResourceFromGitCommand struct {
	Name          string
	EnvironmentID string
	CreatedBy     string
	ConnectionID  string
	GitURL        string
	GitBranch     string
	BuildMode     string // "auto" | "dockerfile"
	GitToken      string // empty for public repos
	ImageTag      string // registry image tag to push to when building
	Ports         []ResourcePortInput
	EnvVars       []ResourceEnvVarInput
}

type CreateResourceFromGitHandler struct {
	resourceRepo domain.ResourceRepository
	buildJobRepo domain.BuildJobRepository
	builder      BuildService
	resolveToken *ResolveSourceConnectionTokenHandler
}

func NewCreateResourceFromGitHandler(
	resourceRepo domain.ResourceRepository,
	buildJobRepo domain.BuildJobRepository,
	builder BuildService,
	resolveToken *ResolveSourceConnectionTokenHandler,
) *CreateResourceFromGitHandler {
	return &CreateResourceFromGitHandler{
		resourceRepo: resourceRepo,
		buildJobRepo: buildJobRepo,
		builder:      builder,
		resolveToken: resolveToken,
	}
}

func (h *CreateResourceFromGitHandler) Handle(ctx context.Context, cmd CreateResourceFromGitCommand) (*domain.Resource, error) {
	ports := make([]domain.ResourcePort, 0, len(cmd.Ports))
	for _, p := range cmd.Ports {
		proto := p.Proto
		if proto == "" {
			proto = "tcp"
		}
		ports = append(ports, domain.ResourcePort{
			HostPort:     p.HostPort,
			InternalPort: p.InternalPort,
			Proto:        proto,
			Label:        p.Label,
		})
	}
	envVars := make([]domain.ResourceEnvVar, 0, len(cmd.EnvVars))
	for _, ev := range cmd.EnvVars {
		envVars = append(envVars, domain.ResourceEnvVar{
			Key:      ev.Key,
			Value:    ev.Value,
			IsSecret: ev.IsSecret,
		})
	}

	branch := strings.TrimSpace(cmd.GitBranch)
	if branch == "" {
		branch = "main"
	}
	mode := strings.TrimSpace(cmd.BuildMode)
	if mode == "" {
		mode = domain.BuildJobModeAuto
	}

	resourceID := newResourceID()

	// Create resource in "created" state — no build is started yet.
	resource, err := h.resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            resourceID,
		Name:          cmd.Name,
		Type:          domain.ResourceTypeApp,
		Image:         "",
		Tag:           "",
		Config:        map[string]any{},
		EnvironmentID: cmd.EnvironmentID,
		CreatedBy:     cmd.CreatedBy,
		SourceType:    domain.ResourceSourceGit,
		GitURL:        cmd.GitURL,
		GitBranch:     branch,
		BuildMode:     mode,
		GitToken:      cmd.GitToken,
		ImageTag:      strings.TrimSpace(cmd.ImageTag),
		ConnectionID:  strings.TrimSpace(cmd.ConnectionID),
		Ports:         ports,
		EnvVars:       envVars,
	})
	if err != nil {
		return nil, fmt.Errorf("create resource from git: %w", err)
	}

	// Transition to "created" status (saved, not yet built).
	if err := h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusCreated, ""); err != nil {
		return nil, fmt.Errorf("set created status: %w", err)
	}

	return h.resourceRepo.GetByID(ctx, resource.ID)
}

// ── StartBuildForResource ─────────────────────────────────────────────────────
// Kicks off a build job for an existing git-based resource.

type StartBuildForResourceCommand struct {
	ResourceID string
	UserID     string
}

type StartBuildForResourceHandler struct {
	resourceRepo domain.ResourceRepository
	buildJobRepo domain.BuildJobRepository
	builder      BuildService
	resolveToken *ResolveSourceConnectionTokenHandler
}

func NewStartBuildForResourceHandler(
	resourceRepo domain.ResourceRepository,
	buildJobRepo domain.BuildJobRepository,
	builder BuildService,
	resolveToken *ResolveSourceConnectionTokenHandler,
) *StartBuildForResourceHandler {
	return &StartBuildForResourceHandler{
		resourceRepo: resourceRepo,
		buildJobRepo: buildJobRepo,
		builder:      builder,
		resolveToken: resolveToken,
	}
}

func (h *StartBuildForResourceHandler) Handle(ctx context.Context, cmd StartBuildForResourceCommand) (*domain.BuildJob, error) {
	resource, err := h.resourceRepo.GetByID(ctx, cmd.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("get resource: %w", err)
	}
	if resource.SourceType != domain.ResourceSourceGit {
		return nil, fmt.Errorf("resource is not a git-based resource")
	}
	if resource.GitURL == "" {
		return nil, fmt.Errorf("resource has no git URL")
	}

	imageTag := strings.TrimSpace(resource.ImageTag)
	if imageTag == "" {
		return nil, fmt.Errorf("resource has no image tag configured")
	}

	branch := resource.GitBranch
	if branch == "" {
		branch = "main"
	}
	mode := resource.BuildMode
	if mode == "" {
		mode = domain.BuildJobModeAuto
	}

	// Resolve git token: prefer connection_id → stored git_token → empty (public)
	gitURL := resource.GitURL
	if strings.TrimSpace(resource.ConnectionID) != "" {
		if h.resolveToken == nil {
			return nil, fmt.Errorf("source connection resolver is not configured")
		}
		resolvedToken, err := h.resolveToken.Handle(ctx, cmd.UserID, resource.ConnectionID)
		if err != nil {
			return nil, fmt.Errorf("resolve source connection token: %w", err)
		}
		if resolvedToken != "" {
			gitURL = injectToken(resource.GitURL, resolvedToken)
		}
	} else if strings.TrimSpace(resource.GitToken) != "" {
		gitURL = injectToken(resource.GitURL, resource.GitToken)
	}

	job, err := domain.NewBuildJob(
		newBuildJobID(),
		gitURL,
		branch,
		mode,
		imageTag,
		resource.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("create build job: %w", err)
	}

	savedJob, err := h.buildJobRepo.Save(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("save build job: %w", err)
	}

	// Transition resource to building
	if err := h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusBuilding, ""); err != nil {
		return nil, fmt.Errorf("set building status: %w", err)
	}

	h.builder.RunAsync(savedJob)

	return savedJob, nil
}

func newResourceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "res_" + hex.EncodeToString(b)
}

// injectToken inserts a token into an HTTPS git URL using the
// "x-access-token:<token>" convention required by GitHub App installation
// tokens (ghs_…) and also compatible with classic PATs (ghp_…).
//
//	https://github.com/user/repo
//	→ https://x-access-token:TOKEN@github.com/user/repo
func injectToken(rawURL, token string) string {
	const https = "https://"
	if strings.HasPrefix(rawURL, https) {
		return https + "x-access-token:" + token + "@" + rawURL[len(https):]
	}
	return rawURL
}
