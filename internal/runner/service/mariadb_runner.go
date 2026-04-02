package service

import (
	"bufio"
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

type MariaDBRunner struct {
	installDir string
}

func NewMariaDBRunner(installDir string) *MariaDBRunner {
	return &MariaDBRunner{installDir: strings.TrimSpace(installDir)}
}

func (r *MariaDBRunner) RunLogicalDump(ctx context.Context, req *model.MariaDBLogicalDumpRequest, writer io.Writer) (string, error) {
	if req == nil {
		return "", fmt.Errorf("mariadb dump request is required")
	}
	version, err := r.resolveVersion(ctx, req)
	if err != nil {
		return "", err
	}
	defaultsFilePath, err := createTempMariaDBDefaultsFile(req.Host, req.Port, req.Username, req.Password)
	if err != nil {
		return "", err
	}
	defer os.Remove(defaultsFilePath)

	binPath, err := runnertools.GetMariaDBExecutable(version, runnertools.MariaDBExecutableDump, r.installDir)
	if err != nil {
		return "", err
	}
	args := []string{
		"--defaults-file=" + defaultsFilePath,
		"--host=" + req.Host,
		"--port=" + strconv.Itoa(req.Port),
		"--user=" + req.Username,
		"--single-transaction",
		"--routines",
		"--quick",
		req.Database,
	}
	cmd := exec.CommandContext(ctx, binPath, args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}

	pipeWriter := mariaDBNopWriteCloser{Writer: writer}
	if strings.EqualFold(req.CompressionType, "gzip") {
		pipeWriter = mariaDBNopWriteCloser{Writer: gzip.NewWriter(writer)}
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start mariadb-dump: %w", err)
	}

	copyErrCh := make(chan error, 1)
	go func() {
		defer pipeWriter.Close()
		_, err := io.Copy(pipeWriter, stdoutPipe)
		if err != nil {
			copyErrCh <- fmt.Errorf("copy mariadb-dump output: %w", err)
			return
		}
		copyErrCh <- nil
	}()

	stderrBytes, _ := io.ReadAll(stderrPipe)
	waitErr := cmd.Wait()
	copyErr := <-copyErrCh
	if waitErr != nil {
		return "", fmt.Errorf("mariadb-dump failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	if copyErr != nil {
		return "", copyErr
	}
	return buildMariaDBArtifactName(req.Database, req.CompressionType), nil
}

func (r *MariaDBRunner) RunLogicalRestore(ctx context.Context, req *model.MariaDBLogicalDumpRequest, reader io.Reader) error {
	if req == nil {
		return fmt.Errorf("mariadb restore request is required")
	}
	version, err := r.resolveVersion(ctx, req)
	if err != nil {
		return err
	}
	defaultsFilePath, err := createTempMariaDBDefaultsFile(req.Host, req.Port, req.Username, req.Password)
	if err != nil {
		return err
	}
	defer os.Remove(defaultsFilePath)

	binPath, err := runnertools.GetMariaDBExecutable(version, runnertools.MariaDBExecutableClient, r.installDir)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, binPath,
		"--defaults-file="+defaultsFilePath,
		"--host="+req.Host,
		fmt.Sprintf("--port=%d", req.Port),
		"--user="+req.Username,
		req.Database,
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
		return fmt.Errorf("start mariadb restore: %w", err)
	}

	copyErrCh := make(chan error, 1)
	go func() {
		defer stdinPipe.Close()
		restoreReader, cleanup, err := prepareMariaDBRestoreReader(reader, req.CompressionType)
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
		return fmt.Errorf("mariadb restore failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	return nil
}

func prepareMariaDBRestoreReader(reader io.Reader, compressionType string) (io.Reader, func() error, error) {
	buffered := bufio.NewReader(reader)
	if !strings.EqualFold(compressionType, "gzip") {
		return buffered, nil, nil
	}
	header, err := buffered.Peek(2)
	if err != nil && err != io.EOF {
		return nil, nil, fmt.Errorf("peek restore artifact header: %w", err)
	}
	if len(header) < 2 || header[0] != 0x1f || header[1] != 0x8b {
		return buffered, nil, nil
	}
	gzipReader, err := gzip.NewReader(buffered)
	if err != nil {
		return nil, nil, fmt.Errorf("open gzip reader: %w", err)
	}
	return gzipReader, gzipReader.Close, nil
}

func (r *MariaDBRunner) resolveVersion(ctx context.Context, req *model.MariaDBLogicalDumpRequest) (string, error) {
	if strings.TrimSpace(req.Version) != "" {
		return req.Version, nil
	}
	return runnertools.DetectMariaDBVersion(ctx, infratools.MariaDBConnectionConfig{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Database: req.Database,
	})
}

type mariaDBNopWriteCloser struct {
	io.Writer
}

func (n mariaDBNopWriteCloser) Close() error {
	if closer, ok := n.Writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func buildMariaDBArtifactName(database string, compressionType string) string {
	base := strings.TrimSpace(database)
	if base == "" {
		base = "mariadb-backup"
	}
	if strings.EqualFold(compressionType, "gzip") {
		return base + ".sql.gz"
	}
	return base + ".sql"
}

func createTempMariaDBDefaultsFile(host string, port int, username string, password string) (string, error) {
	file, err := os.CreateTemp("", "mariadb-runner-*.cnf")
	if err != nil {
		return "", fmt.Errorf("create temp cnf: %w", err)
	}
	defer file.Close()
	content := fmt.Sprintf("[client]\nhost=%s\nport=%d\nuser=%s\npassword=%s\n", host, port, username, password)
	if _, err := file.WriteString(content); err != nil {
		return "", fmt.Errorf("write temp cnf: %w", err)
	}
	if err := os.Chmod(file.Name(), 0o600); err != nil {
		return "", fmt.Errorf("chmod temp cnf: %w", err)
	}
	return file.Name(), nil
}
