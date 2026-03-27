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

type CreateResourceFromGitCommand struct {
	Name          string
	EnvironmentID string
	CreatedBy     string
	GitURL        string
	GitBranch     string
	BuildMode     string // "auto" | "dockerfile"
	GitToken      string // empty for public repos
	ImageTag      string // registry image tag to push to
	Ports         []ResourcePortInput
	EnvVars       []ResourceEnvVarInput
}

type CreateResourceFromGitHandler struct {
	resourceRepo domain.ResourceRepository
	buildJobRepo domain.BuildJobRepository
	builder      BuildService
}

func NewCreateResourceFromGitHandler(
	resourceRepo domain.ResourceRepository,
	buildJobRepo domain.BuildJobRepository,
	builder BuildService,
) *CreateResourceFromGitHandler {
	return &CreateResourceFromGitHandler{resourceRepo: resourceRepo, buildJobRepo: buildJobRepo, builder: builder}
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

	// Create resource in pending_build state; image will be filled after build.
	resource, err := h.resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            resourceID,
		Name:          cmd.Name,
		Type:          domain.ResourceTypeApp,
		Image:         "", // filled by build
		Tag:           "",
		Config:        map[string]any{},
		EnvironmentID: cmd.EnvironmentID,
		CreatedBy:     cmd.CreatedBy,
		SourceType:    domain.ResourceSourceGit,
		GitURL:        cmd.GitURL,
		GitBranch:     branch,
		BuildMode:     mode,
		GitToken:      cmd.GitToken,
		Ports:         ports,
		EnvVars:       envVars,
	})
	if err != nil {
		return nil, fmt.Errorf("create resource from git: %w", err)
	}

	// Transition immediately to pending_build
	if err := h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusPendingBuild, ""); err != nil {
		return nil, fmt.Errorf("set pending_build status: %w", err)
	}

	// Kick off the build job; ResourceID is set so build_service will auto-start when done.
	gitURL := cmd.GitURL
	if cmd.GitToken != "" {
		// Inject token into HTTPS URL: https://token@github.com/...
		gitURL = injectToken(cmd.GitURL, cmd.GitToken)
	}

	job, err := domain.NewBuildJob(
		newBuildJobID(),
		gitURL,
		branch,
		mode,
		cmd.ImageTag,
		resource.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("create build job: %w", err)
	}

	// Persist the build job before running it
	savedJob, err := h.buildJobRepo.Save(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("save build job: %w", err)
	}

	// Transition resource to building
	if err := h.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusBuilding, ""); err != nil {
		return nil, err
	}

	h.builder.RunAsync(savedJob)

	return h.resourceRepo.GetByID(ctx, resource.ID)
}

func newResourceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "res_" + hex.EncodeToString(b)
}

// injectToken inserts a token into an HTTPS git URL.
// e.g. https://github.com/user/repo → https://token@github.com/user/repo
func injectToken(rawURL, token string) string {
	const https = "https://"
	if strings.HasPrefix(rawURL, https) {
		return https + token + "@" + rawURL[len(https):]
	}
	return rawURL
}
