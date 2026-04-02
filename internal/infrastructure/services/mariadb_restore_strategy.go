package services

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type mariaDBRestoreProcess interface {
	StdinPipe() (io.WriteCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

type mariaDBRestoreProcessFactory interface {
	New(ctx context.Context, name string, args ...string) mariaDBRestoreProcess
}

type execMariaDBRestoreProcessFactory struct{}
type execMariaDBRestoreProcess struct{ cmd *exec.Cmd }

func (f *execMariaDBRestoreProcessFactory) New(ctx context.Context, name string, args ...string) mariaDBRestoreProcess {
	return &execMariaDBRestoreProcess{cmd: exec.CommandContext(ctx, name, args...)}
}
func (p *execMariaDBRestoreProcess) StdinPipe() (io.WriteCloser, error) { return p.cmd.StdinPipe() }
func (p *execMariaDBRestoreProcess) StderrPipe() (io.ReadCloser, error) { return p.cmd.StderrPipe() }
func (p *execMariaDBRestoreProcess) Start() error                       { return p.cmd.Start() }
func (p *execMariaDBRestoreProcess) Wait() error                        { return p.cmd.Wait() }

type mariaDBRestoreStrategy struct {
	processFactory mariaDBRestoreProcessFactory
	resolver       mariaDBBinaryResolver
}

func NewMariaDBRestoreStrategy(installDir string) appservices.RestoreStrategy {
	return &mariaDBRestoreStrategy{
		processFactory: &execMariaDBRestoreProcessFactory{},
		resolver:       &defaultMariaDBBinaryResolver{installDir: installDir},
	}
}

func newMariaDBRestoreStrategyWithFactory(factory mariaDBRestoreProcessFactory, resolver mariaDBBinaryResolver) *mariaDBRestoreStrategy {
	return &mariaDBRestoreStrategy{processFactory: factory, resolver: resolver}
}

func (s *mariaDBRestoreStrategy) Execute(ctx context.Context, target *domain.DatabaseSource, backup *domain.Backup, localPath string) error {
	if target == nil || backup == nil {
		return fmt.Errorf("target and backup are required")
	}
	defaultsFilePath, err := createTempMariaDBCnf(target)
	if err != nil {
		return err
	}
	defer os.Remove(defaultsFilePath)

	binPath, err := s.resolver.MariaDB(target.Version)
	if err != nil {
		return err
	}
	slog.Default().Info("mariadb restore strategy start",
		"database", target.DatabaseName,
		"version", target.Version,
		"host", target.Host,
		"port", target.Port,
		"binary_path", binPath,
		"artifact_path", localPath,
	)
	process := s.processFactory.New(ctx, binPath,
		"--defaults-file="+defaultsFilePath,
		"--host="+target.Host,
		fmt.Sprintf("--port=%d", target.Port),
		"--user="+target.Username,
		target.DatabaseName,
	)
	stdinPipe, err := process.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stderrPipe, err := process.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	if err := process.Start(); err != nil {
		return fmt.Errorf("start mariadb restore: %w", err)
	}

	copyErrCh := make(chan error, 1)
	go func() {
		defer stdinPipe.Close()
		if err := streamMariaDBBackupToWriter(localPath, stdinPipe); err != nil {
			copyErrCh <- err
			return
		}
		copyErrCh <- nil
	}()

	stderrBytes, _ := io.ReadAll(stderrPipe)
	waitErr := process.Wait()
	copyErr := <-copyErrCh
	if copyErr != nil {
		return copyErr
	}
	if waitErr != nil {
		return fmt.Errorf("mariadb restore failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	slog.Default().Info("mariadb restore strategy finished",
		"database", target.DatabaseName,
		"version", target.Version,
		"artifact_path", localPath,
	)
	return nil
}

func streamMariaDBBackupToWriter(localPath string, writer io.Writer) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open backup artifact: %w", err)
	}
	defer file.Close()
	var reader io.Reader = file
	if strings.HasSuffix(strings.ToLower(localPath), ".gz") {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("open gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}
	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("stream restore input: %w", err)
	}
	return nil
}
