package service

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	infratools "tango/internal/infrastructure/tools"
	"tango/internal/runner/model"
	runnertools "tango/internal/runner/tools"
)

type PostgresRunner struct {
	installDir string
}

func NewPostgresRunner(installDir string) *PostgresRunner {
	return &PostgresRunner{installDir: strings.TrimSpace(installDir)}
}

func (r *PostgresRunner) RunLogicalDump(ctx context.Context, req *model.PostgresLogicalDumpRequest, writer io.Writer) (string, error) {
	if req == nil {
		return "", fmt.Errorf("postgres dump request is required")
	}
	version, err := r.resolveVersion(ctx, req)
	if err != nil {
		return "", err
	}
	pgPassPath, err := createTempPGPass(req.Host, req.Port, req.Username, req.Password)
	if err != nil {
		return "", err
	}
	defer os.Remove(pgPassPath)

	binPath, err := runnertools.GetPostgresExecutable(version, runnertools.PostgresExecutableDump, r.installDir)
	if err != nil {
		return "", err
	}
	args := []string{
		"-Fc",
		"--no-password",
		"-h", req.Host,
		"-p", strconv.Itoa(req.Port),
		"-U", req.Username,
		"-d", req.Database,
	}
	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Env = append(os.Environ(),
		"PGPASSFILE="+pgPassPath,
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}

	pipeWriter := nopWriteCloser{Writer: writer}
	if strings.EqualFold(req.CompressionType, "gzip") {
		pipeWriter = nopWriteCloser{Writer: gzip.NewWriter(writer)}
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start pg_dump: %w", err)
	}
	copyErrCh := make(chan error, 1)
	go func() {
		defer pipeWriter.Close()
		_, err := io.Copy(pipeWriter, stdoutPipe)
		if err != nil {
			copyErrCh <- fmt.Errorf("copy pg_dump output: %w", err)
			return
		}
		copyErrCh <- nil
	}()
	stderrBytes, _ := io.ReadAll(stderrPipe)
	waitErr := cmd.Wait()
	copyErr := <-copyErrCh
	if waitErr != nil {
		return "", fmt.Errorf("pg_dump failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	if copyErr != nil {
		return "", copyErr
	}
	return buildPostgresArtifactNameFromRequest(req.Database, req.CompressionType), nil
}

func (r *PostgresRunner) RunLogicalRestore(ctx context.Context, req *model.PostgresLogicalDumpRequest, reader io.Reader) error {
	if req == nil {
		return fmt.Errorf("postgres restore request is required")
	}
	version, err := r.resolveVersion(ctx, req)
	if err != nil {
		return err
	}
	pgPassPath, err := createTempPGPass(req.Host, req.Port, req.Username, req.Password)
	if err != nil {
		return err
	}
	defer os.Remove(pgPassPath)

	binPath, err := runnertools.GetPostgresExecutable(version, runnertools.PostgresExecutableRestore, r.installDir)
	if err != nil {
		return err
	}
	args := []string{
		"--no-password",
		"-h", req.Host,
		"-p", strconv.Itoa(req.Port),
		"-U", req.Username,
		"-d", req.Database,
		"--clean",
		"--if-exists",
		"--no-owner",
		"--no-privileges",
	}
	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Env = append(os.Environ(),
		"PGPASSFILE="+pgPassPath,
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pg_restore: %w", err)
	}
	copyErrCh := make(chan error, 1)
	go func() {
		defer stdinPipe.Close()
		restoreReader, cleanup, err := prepareRestoreReader(reader, req.CompressionType)
		if err != nil {
			copyErrCh <- err
			return
		}
		if cleanup != nil {
			defer cleanup()
		}
		_, err = io.Copy(stdinPipe, restoreReader)
		if err != nil {
			copyErrCh <- fmt.Errorf("stream restore input: %w", err)
			return
		}
		copyErrCh <- nil
	}()
	stderrBytes, _ := io.ReadAll(stderrPipe)
	waitErr := cmd.Wait()
	copyErr := <-copyErrCh
	if copyErr != nil {
		return copyErr
	}
	if waitErr != nil {
		return fmt.Errorf("pg_restore failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	return nil
}

func (r *PostgresRunner) resolveVersion(ctx context.Context, req *model.PostgresLogicalDumpRequest) (string, error) {
	if strings.TrimSpace(req.Version) != "" {
		return req.Version, nil
	}
	return runnertools.DetectPostgresVersion(ctx, infratools.PostgresConnectionConfig{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Database: req.Database,
	})
}

func buildPostgresArtifactNameFromRequest(database string, compressionType string) string {
	base := strings.TrimSpace(database)
	if base == "" {
		base = "postgres-backup"
	}
	if strings.EqualFold(compressionType, "gzip") {
		return base + ".dump.gz"
	}
	return base + ".dump"
}

func createTempPGPass(host string, port int, username string, password string) (string, error) {
	file, err := os.CreateTemp("", "postgres-runner-*.pgpass")
	if err != nil {
		return "", fmt.Errorf("create temp pgpass: %w", err)
	}
	defer file.Close()
	content := fmt.Sprintf("%s:%d:*:%s:%s\n", host, port, username, password)
	if _, err := file.WriteString(content); err != nil {
		return "", fmt.Errorf("write temp pgpass: %w", err)
	}
	if err := os.Chmod(file.Name(), 0o600); err != nil {
		return "", fmt.Errorf("chmod temp pgpass: %w", err)
	}
	return file.Name(), nil
}
