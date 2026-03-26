package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
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

// BuildService clones a git repo and calls buildctl to build + push a Docker
// image via a remote BuildKit daemon.
type BuildService struct {
	cfg    BuildConfig
	repo   domain.BuildJobRepository
	logger *slog.Logger
	active sync.Map // jobID → *LogBroadcaster
}

// NewBuildService creates a BuildService.
func NewBuildService(cfg BuildConfig, repo domain.BuildJobRepository, logger *slog.Logger) *BuildService {
	if cfg.BuildTimeout == 0 {
		cfg.BuildTimeout = 20 * time.Minute
	}
	if cfg.CloneTimeout == 0 {
		cfg.CloneTimeout = 5 * time.Minute
	}
	return &BuildService{cfg: cfg, repo: repo, logger: logger}
}

// RunAsync launches the build in a background goroutine.
func (s *BuildService) RunAsync(job *domain.BuildJob) {
	go func() {
		if err := s.run(job); err != nil {
			s.logger.Error("build job failed", "job_id", job.ID, "err", err)
		}
	}()
}

// Subscribe returns a channel that streams log chunks for the given job and an
// unsubscribe func. Returns a nil channel when the job is not currently running
// (already finished or unknown).
func (s *BuildService) Subscribe(jobID string) (<-chan []byte, func()) {
	val, ok := s.active.Load(jobID)
	if !ok {
		return nil, func() {}
	}
	return val.(*LogBroadcaster).Subscribe()
}

func (s *BuildService) run(job *domain.BuildJob) error {
	ctx := context.Background()
	workDir := filepath.Join(s.cfg.WorkspaceDir, job.ID)

	// Register broadcaster so WS clients can subscribe.
	b := newLogBroadcaster()
	s.active.Store(job.ID, b)
	defer func() {
		b.closeAll()
		s.active.Delete(job.ID)
		_ = os.RemoveAll(workDir)
	}()

	log := func(line string) {
		fmt.Fprintln(b, line) //nolint:errcheck
	}

	fail := func(msg string, err error) error {
		errMsg := msg
		if err != nil {
			errMsg = msg + ": " + err.Error()
		}
		log("[ERROR] " + errMsg)
		now := time.Now().UTC()
		job.Status = domain.BuildJobStatusFailed
		job.ErrorMsg = errMsg
		job.Logs = b.Snapshot()
		job.FinishedAt = &now
		if _, updateErr := s.repo.Update(ctx, job); updateErr != nil {
			s.logger.Error("update build job on failure", "job_id", job.ID, "err", updateErr)
		}
		return fmt.Errorf("%s", errMsg)
	}

	updateStatus := func(status domain.BuildJobStatus) {
		job.Status = status
		job.Logs = b.Snapshot()
		if _, err := s.repo.Update(ctx, job); err != nil {
			s.logger.Warn("update build job status", "job_id", job.ID, "status", status, "err", err)
		}
	}

	// Mark started
	now := time.Now().UTC()
	job.StartedAt = &now
	updateStatus(domain.BuildJobStatusCloning)

	// ── 1. Clone ──────────────────────────────────────────────────────────────
	log(fmt.Sprintf("[clone] git clone %s (branch: %s)", job.GitURL, job.GitBranch))
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fail("create workspace dir", err)
	}

	cloneCtx, cloneCancel := context.WithTimeout(ctx, s.cfg.CloneTimeout)
	defer cloneCancel()

	cloneArgs := []string{"clone", "--depth", "1", "--branch", job.GitBranch, job.GitURL, workDir}
	if err := runCmd(cloneCtx, "", b, "git", cloneArgs...); err != nil {
		log("[clone] branch not found, retrying on default branch")
		_ = os.RemoveAll(workDir)
		_ = os.MkdirAll(workDir, 0o755)
		cloneArgs = []string{"clone", "--depth", "1", job.GitURL, workDir}
		if err2 := runCmd(cloneCtx, "", b, "git", cloneArgs...); err2 != nil {
			return fail("git clone failed", err2)
		}
	}

	// ── 2. Detect stack ───────────────────────────────────────────────────────
	updateStatus(domain.BuildJobStatusDetecting)
	stack := DetectStack(workDir)
	log(fmt.Sprintf("[detect] stack detected: %s", stack))

	if stack == StackUnknown {
		return fail("could not detect stack — no known manifest found", nil)
	}

	// ── 3. Generate Dockerfile if needed ─────────────────────────────────────
	if stack != StackDockerfile {
		updateStatus(domain.BuildJobStatusGenerating)
		log(fmt.Sprintf("[generate] creating Dockerfile for %s stack", stack))
		if err := GenerateDockerfile(workDir, stack); err != nil {
			return fail("generate Dockerfile", err)
		}
	}

	// ── 4. Build + push via buildctl ─────────────────────────────────────────
	updateStatus(domain.BuildJobStatusBuilding)
	log(fmt.Sprintf("[build] building image %s", job.ImageTag))

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

	if s.cfg.RegistryUsername != "" && s.cfg.RegistryPassword != "" {
		configDir, err := writeDockerConfig(s.cfg.RegistryHost, s.cfg.RegistryUsername, s.cfg.RegistryPassword)
		if err != nil {
			return fail("write docker config for registry auth", err)
		}
		defer os.RemoveAll(configDir)
		buildArgs = append([]string{"--config", filepath.Join(configDir, "config.json")}, buildArgs...)
	}

	if err := runCmd(buildCtx, "", b, "buildctl", buildArgs...); err != nil {
		return fail("buildctl build failed", err)
	}

	// ── 5. Done ───────────────────────────────────────────────────────────────
	log(fmt.Sprintf("[done] image pushed: %s", job.ImageTag))
	done := time.Now().UTC()
	job.Status = domain.BuildJobStatusDone
	job.FinishedAt = &done
	job.Logs = b.Snapshot()
	if _, err := s.repo.Update(ctx, job); err != nil {
		s.logger.Error("update build job on success", "job_id", job.ID, "err", err)
	}
	return nil
}

// runCmd executes a command and streams stdout+stderr to w in real-time.
func runCmd(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

// writeDockerConfig creates a temp dir with a minimal docker config.json.
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
			registryKey: map[string]any{"auth": authToken},
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
