package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tango/internal/domain"
)

// BuildConfig holds all settings needed to run the build service.
type BuildConfig struct {
	// BuildKitHost is the BuildKit daemon address, e.g. tcp://buildkitd:1234
	BuildKitHost string
	// WorkspaceDir is the base directory for cloned repos, e.g. /workspace/jobs
	WorkspaceDir string
	// RegistryHost is the registry to push images to, e.g. ghcr.io or docker.io
	RegistryHost string
	// RegistryUsername for docker login
	RegistryUsername string
	// RegistryPassword for docker login
	RegistryPassword string
	// BuildTimeout is the maximum time allowed for a single build
	BuildTimeout time.Duration
	// CloneTimeout is the maximum time allowed for a git clone
	CloneTimeout time.Duration
}

type buildService struct {
	cfg    BuildConfig
	repo   domain.BuildJobRepository
	logger *slog.Logger
}

// NewBuildService creates a BuildService that clones a git repo and calls
// buildctl to build + push a Docker image via a remote BuildKit daemon.
func NewBuildService(cfg BuildConfig, repo domain.BuildJobRepository, logger *slog.Logger) *buildService {
	if cfg.BuildTimeout == 0 {
		cfg.BuildTimeout = 20 * time.Minute
	}
	if cfg.CloneTimeout == 0 {
		cfg.CloneTimeout = 5 * time.Minute
	}
	return &buildService{cfg: cfg, repo: repo, logger: logger}
}

// RunAsync launches the build in a background goroutine. Satisfies command.BuildService.
func (s *buildService) RunAsync(job *domain.BuildJob) {
	go func() {
		if err := s.run(job); err != nil {
			s.logger.Error("build job failed", "job_id", job.ID, "err", err)
		}
	}()
}

func (s *buildService) run(job *domain.BuildJob) error {
	ctx := context.Background()
	workDir := filepath.Join(s.cfg.WorkspaceDir, job.ID)

	defer func() {
		// Best-effort cleanup
		_ = os.RemoveAll(workDir)
	}()

	var logBuf strings.Builder

	appendLog := func(line string) {
		logBuf.WriteString(line)
		logBuf.WriteByte('\n')
	}

	fail := func(msg string, err error) error {
		errMsg := msg
		if err != nil {
			errMsg = msg + ": " + err.Error()
		}
		appendLog("[ERROR] " + errMsg)
		now := time.Now().UTC()
		job.Status = domain.BuildJobStatusFailed
		job.ErrorMsg = errMsg
		job.Logs = logBuf.String()
		job.FinishedAt = &now
		if _, updateErr := s.repo.Update(ctx, job); updateErr != nil {
			s.logger.Error("update build job on failure", "job_id", job.ID, "err", updateErr)
		}
		s.logger.Error("build job failed", "job_id", job.ID, "error", errMsg, "logs", logBuf.String())
		return fmt.Errorf("%s", errMsg)
	}

	updateStatus := func(status domain.BuildJobStatus) {
		job.Status = status
		job.Logs = logBuf.String()
		if _, err := s.repo.Update(ctx, job); err != nil {
			s.logger.Warn("update build job status", "job_id", job.ID, "status", status, "err", err)
		}
	}

	// Mark started
	now := time.Now().UTC()
	job.StartedAt = &now
	updateStatus(domain.BuildJobStatusCloning)

	// ── 1. Clone ──────────────────────────────────────────────────────────────
	appendLog(fmt.Sprintf("[clone] git clone %s (branch: %s)", job.GitURL, job.GitBranch))
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fail("create workspace dir", err)
	}

	cloneCtx, cloneCancel := context.WithTimeout(ctx, s.cfg.CloneTimeout)
	defer cloneCancel()

	cloneArgs := []string{"clone", "--depth", "1", "--branch", job.GitBranch, job.GitURL, workDir}
	cloneOut, err := runCmd(cloneCtx, "", "git", cloneArgs...)
	appendLog(cloneOut)
	if err != nil {
		// Retry without --branch (default branch)
		appendLog("[clone] branch not found, retrying on default branch")
		_ = os.RemoveAll(workDir)
		_ = os.MkdirAll(workDir, 0o755)
		cloneArgs = []string{"clone", "--depth", "1", job.GitURL, workDir}
		cloneOut2, err2 := runCmd(cloneCtx, "", "git", cloneArgs...)
		appendLog(cloneOut2)
		if err2 != nil {
			return fail("git clone failed", err2)
		}
	}

	// ── 2. Detect stack ───────────────────────────────────────────────────────
	updateStatus(domain.BuildJobStatusDetecting)
	stack := DetectStack(workDir)
	appendLog(fmt.Sprintf("[detect] stack detected: %s", stack))

	if stack == StackUnknown {
		return fail("could not detect stack — no known manifest found", nil)
	}

	// ── 3. Generate Dockerfile if needed ─────────────────────────────────────
	if stack != StackDockerfile {
		updateStatus(domain.BuildJobStatusGenerating)
		appendLog(fmt.Sprintf("[generate] creating Dockerfile for %s stack", stack))
		if err := GenerateDockerfile(workDir, stack); err != nil {
			return fail("generate Dockerfile", err)
		}
	}

	// ── 4. Build + push via buildctl ─────────────────────────────────────────
	updateStatus(domain.BuildJobStatusBuilding)
	appendLog(fmt.Sprintf("[build] building image %s", job.ImageTag))

	buildCtx, buildCancel := context.WithTimeout(ctx, s.cfg.BuildTimeout)
	defer buildCancel()

	buildArgs := []string{
		"--addr", s.cfg.BuildKitHost,
		"build",
		"--frontend", "dockerfile.v0",
		"--local", "context=" + workDir,
		"--local", "dockerfile=" + workDir,
		"--output", fmt.Sprintf("type=image,name=%s,push=true", job.ImageTag),
	}

	// Write a docker config.json if registry credentials are provided so
	// buildctl can authenticate when pushing.
	if s.cfg.RegistryUsername != "" && s.cfg.RegistryPassword != "" {
		configDir, err := writeDockerConfig(s.cfg.RegistryHost, s.cfg.RegistryUsername, s.cfg.RegistryPassword)
		if err != nil {
			return fail("write docker config for registry auth", err)
		}
		defer os.RemoveAll(configDir)
		buildArgs = append([]string{"--config", filepath.Join(configDir, "config.json")}, buildArgs...)
	}

	buildOut, err := runCmd(buildCtx, "", "buildctl", buildArgs...)
	appendLog(buildOut)
	if err != nil {
		return fail("buildctl build failed", err)
	}

	// ── 5. Done ───────────────────────────────────────────────────────────────
	done := time.Now().UTC()
	job.Status = domain.BuildJobStatusDone
	job.FinishedAt = &done
	job.Logs = logBuf.String()
	if _, err := s.repo.Update(ctx, job); err != nil {
		s.logger.Error("update build job on success", "job_id", job.ID, "err", err)
	}
	appendLog(fmt.Sprintf("[done] image pushed: %s", job.ImageTag))
	return nil
}

// runCmd executes a command and returns combined stdout+stderr output.
func runCmd(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// writeDockerConfig creates a temp dir with a minimal docker config.json for
// registry authentication used by buildctl --config.
func writeDockerConfig(host, username, password string) (string, error) {
	dir, err := os.MkdirTemp("", "buildkit-auth-*")
	if err != nil {
		return "", err
	}

	registryKey := host
	if registryKey == "" {
		registryKey = "https://index.docker.io/v1/"
	}
	authToken := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	cfg := map[string]any{
		"auths": map[string]any{
			registryKey: map[string]any{
				"auth": authToken,
			},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), data, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}
