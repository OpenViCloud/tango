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

type mariaDBDumpProcess interface {
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

type mariaDBDumpProcessFactory interface {
	New(ctx context.Context, name string, args ...string) mariaDBDumpProcess
}

type execMariaDBDumpProcessFactory struct{}
type execMariaDBDumpProcess struct{ cmd *exec.Cmd }

func (f *execMariaDBDumpProcessFactory) New(ctx context.Context, name string, args ...string) mariaDBDumpProcess {
	return &execMariaDBDumpProcess{cmd: exec.CommandContext(ctx, name, args...)}
}
func (p *execMariaDBDumpProcess) StdoutPipe() (io.ReadCloser, error) { return p.cmd.StdoutPipe() }
func (p *execMariaDBDumpProcess) StderrPipe() (io.ReadCloser, error) { return p.cmd.StderrPipe() }
func (p *execMariaDBDumpProcess) Start() error                       { return p.cmd.Start() }
func (p *execMariaDBDumpProcess) Wait() error                        { return p.cmd.Wait() }

type mariaDBLogicalDumpStrategy struct {
	processFactory mariaDBDumpProcessFactory
	resolver       mariaDBBinaryResolver
}

type mariaDBBinaryResolver interface {
	MariaDBDump(version string) (string, error)
	MariaDB(version string) (string, error)
}

type defaultMariaDBBinaryResolver struct {
	installDir string
}

func NewMariaDBLogicalDumpStrategy(installDir string) appservices.BackupStrategy {
	return &mariaDBLogicalDumpStrategy{
		processFactory: &execMariaDBDumpProcessFactory{},
		resolver:       &defaultMariaDBBinaryResolver{installDir: installDir},
	}
}

func newMariaDBLogicalDumpStrategyWithFactory(factory mariaDBDumpProcessFactory, resolver mariaDBBinaryResolver) *mariaDBLogicalDumpStrategy {
	return &mariaDBLogicalDumpStrategy{processFactory: factory, resolver: resolver}
}

func (s *mariaDBLogicalDumpStrategy) Execute(ctx context.Context, source *domain.DatabaseSource, config *domain.BackupConfig) (*appservices.BackupArtifact, error) {
	if source == nil || config == nil {
		return nil, fmt.Errorf("source and config are required")
	}
	defaultsFilePath, err := createTempMariaDBCnf(source)
	if err != nil {
		return nil, err
	}
	defer os.Remove(defaultsFilePath)

	fileName := buildMariaDBBackupArtifactName(source, config)
	outputPath := filepath.Join(os.TempDir(), "tango-backup-"+fileName)
	slog.Default().Info("mariadb dump strategy start",
		"database", source.DatabaseName,
		"version", source.Version,
		"host", source.Host,
		"port", source.Port,
		"file_name", fileName,
		"compression", config.CompressionType,
	)
	if err := runMariaDBDumpToFile(ctx, s.processFactory, s.resolver, source, config, defaultsFilePath, outputPath); err != nil {
		_ = os.Remove(outputPath)
		return nil, err
	}
	slog.Default().Info("mariadb dump strategy finished",
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
			"mariadb_version":  source.Version,
			"tool":             "mariadb-dump",
		},
	}, nil
}

func buildMariaDBBackupArtifactName(source *domain.DatabaseSource, config *domain.BackupConfig) string {
	base := strings.TrimSpace(source.DatabaseName)
	if base == "" {
		base = "mariadb-backup"
	}
	if config.CompressionType == domain.BackupCompressionGzip {
		return base + ".sql.gz"
	}
	return base + ".sql"
}

func runMariaDBDumpToFile(ctx context.Context, factory mariaDBDumpProcessFactory, resolver mariaDBBinaryResolver, source *domain.DatabaseSource, config *domain.BackupConfig, defaultsFile string, outputPath string) error {
	binPath, err := resolver.MariaDBDump(source.Version)
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
		return fmt.Errorf("start mariadb-dump: %w", err)
	}
	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(writer, stdoutPipe)
		if err != nil {
			copyErrCh <- fmt.Errorf("copy mariadb-dump output: %w", err)
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
		return fmt.Errorf("mariadb-dump failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	if copyErr != nil {
		return copyErr
	}
	return nil
}

func (r *defaultMariaDBBinaryResolver) MariaDBDump(version string) (string, error) {
	return tools.GetMariaDBExecutable(version, tools.MariaDBExecutableDump, r.installDir)
}

func (r *defaultMariaDBBinaryResolver) MariaDB(version string) (string, error) {
	return tools.GetMariaDBExecutable(version, tools.MariaDBExecutableClient, r.installDir)
}

func createTempMariaDBCnf(source *domain.DatabaseSource) (string, error) {
	file, err := os.CreateTemp("", "mariadb-backup-*.cnf")
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
