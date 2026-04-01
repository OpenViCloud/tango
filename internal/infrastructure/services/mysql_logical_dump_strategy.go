package services

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
	"tango/internal/infrastructure/tools"
)

type dumpProcess interface {
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

type dumpProcessFactory interface {
	New(ctx context.Context, name string, args ...string) dumpProcess
}

type execDumpProcessFactory struct{}
type execDumpProcess struct{ cmd *exec.Cmd }

func (f *execDumpProcessFactory) New(ctx context.Context, name string, args ...string) dumpProcess {
	return &execDumpProcess{cmd: exec.CommandContext(ctx, name, args...)}
}
func (p *execDumpProcess) StdoutPipe() (io.ReadCloser, error) { return p.cmd.StdoutPipe() }
func (p *execDumpProcess) StderrPipe() (io.ReadCloser, error) { return p.cmd.StderrPipe() }
func (p *execDumpProcess) Start() error                       { return p.cmd.Start() }
func (p *execDumpProcess) Wait() error                        { return p.cmd.Wait() }

type mySQLLogicalDumpStrategy struct {
	processFactory dumpProcessFactory
	resolver       mySQLBinaryResolver
}

type mySQLBinaryResolver interface {
	Mysqldump(version string) (string, error)
	Mysql(version string) (string, error)
}

type defaultMySQLBinaryResolver struct {
	installDir string
}

func NewMySQLLogicalDumpStrategy(installDir string) appservices.BackupStrategy {
	return &mySQLLogicalDumpStrategy{
		processFactory: &execDumpProcessFactory{},
		resolver:       &defaultMySQLBinaryResolver{installDir: installDir},
	}
}

func newMySQLLogicalDumpStrategyWithFactory(factory dumpProcessFactory, resolver mySQLBinaryResolver) *mySQLLogicalDumpStrategy {
	return &mySQLLogicalDumpStrategy{processFactory: factory, resolver: resolver}
}

func (s *mySQLLogicalDumpStrategy) Execute(ctx context.Context, source *domain.DatabaseSource, config *domain.BackupConfig) (*appservices.BackupArtifact, error) {
	if source == nil || config == nil {
		return nil, fmt.Errorf("source and config are required")
	}
	myCnfPath, err := createTempMyCnf(source)
	if err != nil {
		return nil, err
	}
	defer os.Remove(myCnfPath)

	fileName := buildMySQLArtifactName(source, config)
	outputPath := filepath.Join(os.TempDir(), "tango-backup-"+fileName)
	slog.Default().Info("mysql dump strategy start",
		"database", source.DatabaseName,
		"version", source.Version,
		"host", source.Host,
		"port", source.Port,
		"file_name", fileName,
		"compression", config.CompressionType,
	)
	if err := runMySQLDumpToFile(ctx, s.processFactory, s.resolver, source, config, myCnfPath, outputPath); err != nil {
		_ = os.Remove(outputPath)
		return nil, err
	}
	slog.Default().Info("mysql dump strategy finished",
		"database", source.DatabaseName,
		"version", source.Version,
		"output_path", outputPath,
	)
	return &appservices.BackupArtifact{
		FileName:  fileName,
		LocalPath: outputPath,
		Metadata: map[string]any{
			"db_type":          source.DBType,
			"database_name":    source.DatabaseName,
			"backup_method":    config.BackupMethod,
			"compression_type": config.CompressionType,
			"mysql_version":    source.Version,
			"tool":             "mysqldump",
		},
	}, nil
}

func buildMySQLArtifactName(source *domain.DatabaseSource, config *domain.BackupConfig) string {
	base := strings.TrimSpace(source.DatabaseName)
	if base == "" {
		base = "mysql-backup"
	}
	if config.CompressionType == domain.BackupCompressionGzip {
		return base + ".sql.gz"
	}
	return base + ".sql"
}

func runMySQLDumpToFile(ctx context.Context, factory dumpProcessFactory, resolver mySQLBinaryResolver, source *domain.DatabaseSource, config *domain.BackupConfig, defaultsFile string, outputPath string) error {
	binPath, err := resolver.Mysqldump(source.Version)
	if err != nil {
		return err
	}
	args := []string{
		"--defaults-file=" + defaultsFile,
		"--host=" + source.Host,
		"--port=" + strconv.Itoa(source.Port),
		"--user=" + source.Username,
		"--single-transaction",
		"--routines",
		"--set-gtid-purged=OFF",
		"--quick",
		source.DatabaseName,
	}
	process := factory.New(ctx, binPath, args...)
	stdoutPipe, err := process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := process.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	var writer io.WriteCloser = outFile
	if config.CompressionType == domain.BackupCompressionGzip {
		gzipWriter := gzip.NewWriter(outFile)
		writer = gzipWriter
	}
	if err := process.Start(); err != nil {
		return fmt.Errorf("start mysqldump: %w", err)
	}
	slog.Default().Info("mysqldump process started",
		"database", source.DatabaseName,
		"version", source.Version,
		"binary_path", binPath,
		"output_path", outputPath,
	)
	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(writer, stdoutPipe)
		if err != nil {
			copyErrCh <- fmt.Errorf("copy mysqldump output: %w", err)
			return
		}
		if err := writer.Close(); err != nil {
			copyErrCh <- fmt.Errorf("close dump writer: %w", err)
			return
		}
		copyErrCh <- nil
	}()

	stderrBytes, _ := io.ReadAll(stderrPipe)
	waitErr := process.Wait()
	copyErr := <-copyErrCh
	if waitErr != nil {
		return fmt.Errorf("mysqldump failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	if copyErr != nil {
		return copyErr
	}
	return nil
}

func (r *defaultMySQLBinaryResolver) Mysqldump(version string) (string, error) {
	return tools.GetMySQLExecutable(version, tools.MySQLExecutableDump, r.installDir)
}

func (r *defaultMySQLBinaryResolver) Mysql(version string) (string, error) {
	return tools.GetMySQLExecutable(version, tools.MySQLExecutableClient, r.installDir)
}

func createTempMyCnf(source *domain.DatabaseSource) (string, error) {
	file, err := os.CreateTemp("", "mysql-backup-*.cnf")
	if err != nil {
		return "", fmt.Errorf("create temp cnf: %w", err)
	}
	defer file.Close()
	content := fmt.Sprintf("[client]\nhost=%s\nport=%d\nuser=%s\npassword=%s\n", source.Host, source.Port, source.Username, source.PasswordEncrypted)
	if _, err := file.WriteString(content); err != nil {
		return "", fmt.Errorf("write temp cnf: %w", err)
	}
	if err := os.Chmod(file.Name(), 0o600); err != nil {
		return "", fmt.Errorf("chmod temp cnf: %w", err)
	}
	return file.Name(), nil
}
