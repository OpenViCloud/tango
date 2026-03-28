package services

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// ResourceAutoStarter is called by BuildService after a successful build when
// the job carries a ResourceID. It updates the resource image and starts it.
type ResourceAutoStarter interface {
	StartAfterBuild(ctx context.Context, resourceID, imageTag string) error
}

// BuildService clones a git repo (or extracts an uploaded archive) and calls
// buildctl to build + push a Docker image via a remote BuildKit daemon.
type BuildService struct {
	cfg           BuildConfig
	repo          domain.BuildJobRepository
	logger        *slog.Logger
	active        sync.Map // jobID → *LogBroadcaster
	autoStarter   ResourceAutoStarter // optional; called when job.ResourceID != ""
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

// SetResourceAutoStarter injects the auto-starter after construction (avoids
// circular dependency at wire-up time).
func (s *BuildService) SetResourceAutoStarter(rs ResourceAutoStarter) {
	s.autoStarter = rs
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
// unsubscribe func. Returns a nil channel when the job is not currently running.
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

	now := time.Now().UTC()
	job.StartedAt = &now
	updateStatus(domain.BuildJobStatusCloning)

	// ── 1. Prepare source ─────────────────────────────────────────────────────
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fail("create workspace dir", err)
	}

	if job.SourceType == domain.BuildJobSourceUpload {
		log(fmt.Sprintf("[extract] extracting %s", job.ArchiveName))
		if err := extractArchive(job.ArchivePath, workDir); err != nil {
			return fail("extract archive", err)
		}
		_ = os.Remove(job.ArchivePath) // clean up temp file
	} else {
		// git clone
		log(fmt.Sprintf("[clone] git clone %s (branch: %s)", job.GitURL, job.GitBranch))
		cloneCtx, cloneCancel := context.WithTimeout(ctx, s.cfg.CloneTimeout)
		defer cloneCancel()

		cloneArgs := []string{"clone", "--depth", "1", "--single-branch", "--branch", job.GitBranch, job.GitURL, workDir}
		if err := runGit(cloneCtx, "", b, cloneArgs...); err != nil {
			log("[clone] branch not found, retrying on default branch")
			_ = os.RemoveAll(workDir)
			_ = os.MkdirAll(workDir, 0o755)
			cloneArgs = []string{"clone", "--depth", "1", "--single-branch", job.GitURL, workDir}
			if err2 := runGit(cloneCtx, "", b, cloneArgs...); err2 != nil {
				return fail("git clone failed", err2)
			}
		}
	}

	// ── 2. Detect stack (skip if build_mode = dockerfile) ─────────────────────
	if job.BuildMode != domain.BuildJobModeDockerfile {
		updateStatus(domain.BuildJobStatusDetecting)
		stack := DetectStack(workDir)
		log(fmt.Sprintf("[detect] stack detected: %s", stack))

		if stack == StackUnknown {
			return fail("could not detect stack — no known manifest found", nil)
		}

		// ── 3. Generate Dockerfile if needed ──────────────────────────────────
		if stack != StackDockerfile {
			updateStatus(domain.BuildJobStatusGenerating)
			log(fmt.Sprintf("[generate] creating Dockerfile for %s stack", stack))
			if err := GenerateDockerfile(workDir, stack); err != nil {
				return fail("generate Dockerfile", err)
			}
		}
	} else {
		log("[build] using existing Dockerfile")
	}

	// ── 4. Build + push via buildctl ──────────────────────────────────────────
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

	// ── 6. Auto-start resource if linked ──────────────────────────────────────
	if job.ResourceID != "" && s.autoStarter != nil {
		log(fmt.Sprintf("[deploy] starting resource %s", job.ResourceID))
		if err := s.autoStarter.StartAfterBuild(ctx, job.ResourceID, job.ImageTag); err != nil {
			s.logger.Error("auto-start resource after build failed", "resource_id", job.ResourceID, "err", err)
			log(fmt.Sprintf("[deploy] failed to start resource: %s", err.Error()))
		}
	}

	return nil
}

// ── archive extraction ────────────────────────────────────────────────────────

func extractArchive(archivePath, destDir string) error {
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(archivePath, destDir)
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive format: %s", filepath.Ext(archivePath))
	}
}

func extractTarGz(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	// First pass: collect entry names to detect common top-level dir.
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		names = append(names, hdr.Name)
	}
	gz.Close()
	topDir := detectTopDir(names)

	// Second pass: extract.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	gz2, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz2.Close()
	tr2 := tar.NewReader(gz2)

	cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
	for {
		hdr, err := tr2.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		rel := hdr.Name
		if topDir != "" {
			rel = strings.TrimPrefix(rel, topDir)
		}
		if rel == "" || rel == "." {
			continue
		}
		target := filepath.Join(destDir, filepath.FromSlash(rel))
		if !strings.HasPrefix(target, cleanDest) {
			continue // skip path traversal attempts
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr2)
			out.Close()
			if copyErr != nil {
				return copyErr
			}
		}
	}
	return nil
}

func extractZip(src, destDir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	topDir := detectTopDir(names)

	cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
	for _, f := range r.File {
		rel := f.Name
		if topDir != "" {
			rel = strings.TrimPrefix(rel, topDir)
		}
		if rel == "" || rel == "." {
			continue
		}
		target := filepath.Join(destDir, filepath.FromSlash(rel))
		if !strings.HasPrefix(target, cleanDest) {
			continue
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// detectTopDir returns the common top-level directory prefix shared by all
// entries, so it can be stripped during extraction (e.g. "repo-main/").
func detectTopDir(entries []string) string {
	if len(entries) == 0 {
		return ""
	}
	var top string
	for _, e := range entries {
		idx := strings.Index(e, "/")
		if idx < 0 {
			return "" // entry at root level — nothing to strip
		}
		dir := e[:idx+1]
		if top == "" {
			top = dir
		} else if top != dir {
			return ""
		}
	}
	return top
}

// ── helpers ───────────────────────────────────────────────────────────────────

func runCmd(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

// runGit runs a git command with GIT_TERMINAL_PROMPT=0 so git never hangs
// waiting for a password prompt — it fails immediately on auth errors instead.
func runGit(ctx context.Context, dir string, w io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	return cmd.Run()
}

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
