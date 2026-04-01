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

type restoreProcess interface {
	StdinPipe() (io.WriteCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

type restoreProcessFactory interface {
	New(ctx context.Context, name string, args ...string) restoreProcess
}

type execRestoreProcessFactory struct{}
type execRestoreProcess struct{ cmd *exec.Cmd }

func (f *execRestoreProcessFactory) New(ctx context.Context, name string, args ...string) restoreProcess {
	return &execRestoreProcess{cmd: exec.CommandContext(ctx, name, args...)}
}
func (p *execRestoreProcess) StdinPipe() (io.WriteCloser, error) { return p.cmd.StdinPipe() }
func (p *execRestoreProcess) StderrPipe() (io.ReadCloser, error) { return p.cmd.StderrPipe() }
func (p *execRestoreProcess) Start() error                       { return p.cmd.Start() }
func (p *execRestoreProcess) Wait() error                        { return p.cmd.Wait() }

type mySQLRestoreStrategy struct {
	processFactory restoreProcessFactory
	resolver       mySQLBinaryResolver
}

func NewMySQLRestoreStrategy(installDir string) appservices.RestoreStrategy {
	return &mySQLRestoreStrategy{
		processFactory: &execRestoreProcessFactory{},
		resolver:       &defaultMySQLBinaryResolver{installDir: installDir},
	}
}

func newMySQLRestoreStrategyWithFactory(factory restoreProcessFactory, resolver mySQLBinaryResolver) *mySQLRestoreStrategy {
	return &mySQLRestoreStrategy{processFactory: factory, resolver: resolver}
}

func (s *mySQLRestoreStrategy) Execute(ctx context.Context, target *domain.DatabaseSource, backup *domain.Backup, localPath string) error {
	if target == nil || backup == nil {
		return fmt.Errorf("target and backup are required")
	}
	myCnfPath, err := createTempMyCnf(target)
	if err != nil {
		return err
	}
	defer os.Remove(myCnfPath)

	binPath, err := s.resolver.Mysql(target.Version)
	if err != nil {
		return err
	}
	slog.Default().Info("mysql restore strategy start",
		"database", target.DatabaseName,
		"version", target.Version,
		"host", target.Host,
		"port", target.Port,
		"binary_path", binPath,
		"artifact_path", localPath,
	)
	process := s.processFactory.New(ctx, binPath,
		"--defaults-file="+myCnfPath,
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
		return fmt.Errorf("start mysql restore: %w", err)
	}

	copyErrCh := make(chan error, 1)
	go func() {
		defer stdinPipe.Close()
		if err := streamBackupToWriter(localPath, stdinPipe); err != nil {
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
		return fmt.Errorf("mysql restore failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	slog.Default().Info("mysql restore strategy finished",
		"database", target.DatabaseName,
		"version", target.Version,
		"artifact_path", localPath,
	)
	return nil
}

func streamBackupToWriter(localPath string, writer io.Writer) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open backup artifact: %w", err)
	}
	defer f.Close()
	var reader io.Reader = f
	if strings.HasSuffix(strings.ToLower(localPath), ".gz") {
		gzipReader, err := gzip.NewReader(f)
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
