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

type MySQLRunner struct {
	installDir string
}

func NewMySQLRunner(installDir string) *MySQLRunner {
	return &MySQLRunner{installDir: strings.TrimSpace(installDir)}
}

func (r *MySQLRunner) RunLogicalDump(ctx context.Context, req *model.MySQLLogicalDumpRequest, writer io.Writer) (string, error) {
	if req == nil {
		return "", fmt.Errorf("mysql dump request is required")
	}
	version, err := r.resolveVersion(ctx, req)
	if err != nil {
		return "", err
	}
	myCnfPath, err := createTempMyCnf(req.Host, req.Port, req.Username, req.Password)
	if err != nil {
		return "", err
	}
	defer os.Remove(myCnfPath)

	binPath, err := runnertools.GetMySQLExecutable(version, runnertools.MySQLExecutableDump, r.installDir)
	if err != nil {
		return "", err
	}
	args := []string{
		"--defaults-file=" + myCnfPath,
		"--host=" + req.Host,
		"--port=" + strconv.Itoa(req.Port),
		"--user=" + req.Username,
		"--single-transaction",
		"--routines",
		"--set-gtid-purged=OFF",
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

	pipeWriter := nopWriteCloser{Writer: writer}
	if strings.EqualFold(req.CompressionType, "gzip") {
		pipeWriter = nopWriteCloser{Writer: gzip.NewWriter(writer)}
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start mysqldump: %w", err)
	}

	copyErrCh := make(chan error, 1)
	go func() {
		defer pipeWriter.Close()
		_, err := io.Copy(pipeWriter, stdoutPipe)
		if err != nil {
			copyErrCh <- fmt.Errorf("copy mysqldump output: %w", err)
			return
		}
		copyErrCh <- nil
	}()

	stderrBytes, _ := io.ReadAll(stderrPipe)
	waitErr := cmd.Wait()
	copyErr := <-copyErrCh
	if waitErr != nil {
		return "", fmt.Errorf("mysqldump failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	if copyErr != nil {
		return "", copyErr
	}
	return buildArtifactName(req.Database, req.CompressionType), nil
}

func (r *MySQLRunner) RunLogicalRestore(ctx context.Context, req *model.MySQLLogicalDumpRequest, reader io.Reader) error {
	if req == nil {
		return fmt.Errorf("mysql restore request is required")
	}
	version, err := r.resolveVersion(ctx, req)
	if err != nil {
		return err
	}
	myCnfPath, err := createTempMyCnf(req.Host, req.Port, req.Username, req.Password)
	if err != nil {
		return err
	}
	defer os.Remove(myCnfPath)

	binPath, err := runnertools.GetMySQLExecutable(version, runnertools.MySQLExecutableClient, r.installDir)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, binPath,
		"--defaults-file="+myCnfPath,
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
		return fmt.Errorf("start mysql restore: %w", err)
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
		return fmt.Errorf("mysql restore failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	return nil
}

func prepareRestoreReader(reader io.Reader, compressionType string) (io.Reader, func() error, error) {
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

func (r *MySQLRunner) resolveVersion(ctx context.Context, req *model.MySQLLogicalDumpRequest) (string, error) {
	if strings.TrimSpace(req.Version) != "" {
		return req.Version, nil
	}
	return runnertools.DetectMySQLVersion(ctx, infratools.MySQLConnectionConfig{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Database: req.Database,
	})
}

type nopWriteCloser struct {
	io.Writer
}

func (n nopWriteCloser) Close() error {
	if closer, ok := n.Writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func buildArtifactName(database string, compressionType string) string {
	base := strings.TrimSpace(database)
	if base == "" {
		base = "mysql-backup"
	}
	if strings.EqualFold(compressionType, "gzip") {
		return base + ".sql.gz"
	}
	return base + ".sql"
}

func createTempMyCnf(host string, port int, username string, password string) (string, error) {
	file, err := os.CreateTemp("", "mysql-runner-*.cnf")
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
