package services

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type fakePostgresBackupRunnerClient struct {
	dumpArtifact    *appservices.BackupRunnerArtifact
	dumpErr         error
	restoreErr      error
	restoreReq      *appservices.PostgresLogicalRestoreRequest
	restoreContents []byte
}

func (c *fakePostgresBackupRunnerClient) RunMySQLLogicalDump(_ context.Context, _ *appservices.MySQLLogicalDumpRequest, _ io.Writer) (*appservices.BackupRunnerArtifact, error) {
	return nil, nil
}

func (c *fakePostgresBackupRunnerClient) RunMySQLLogicalRestore(_ context.Context, _ *appservices.MySQLLogicalRestoreRequest, _ io.Reader) error {
	return nil
}

func (c *fakePostgresBackupRunnerClient) RunMariaDBLogicalDump(_ context.Context, _ *appservices.MariaDBLogicalDumpRequest, _ io.Writer) (*appservices.BackupRunnerArtifact, error) {
	return nil, nil
}

func (c *fakePostgresBackupRunnerClient) RunMariaDBLogicalRestore(_ context.Context, _ *appservices.MariaDBLogicalRestoreRequest, _ io.Reader) error {
	return nil
}

func (c *fakePostgresBackupRunnerClient) RunMongoLogicalDump(_ context.Context, _ *appservices.MongoLogicalDumpRequest, _ io.Writer) (*appservices.BackupRunnerArtifact, error) {
	return nil, nil
}

func (c *fakePostgresBackupRunnerClient) RunMongoLogicalRestore(_ context.Context, _ *appservices.MongoLogicalRestoreRequest, _ io.Reader) error {
	return nil
}

func (c *fakePostgresBackupRunnerClient) RunPostgresLogicalDump(_ context.Context, _ *appservices.PostgresLogicalDumpRequest, writer io.Writer) (*appservices.BackupRunnerArtifact, error) {
	if c.dumpErr != nil {
		return nil, c.dumpErr
	}
	if _, err := writer.Write([]byte("postgres-dump-data")); err != nil {
		return nil, err
	}
	return c.dumpArtifact, nil
}

func (c *fakePostgresBackupRunnerClient) RunPostgresLogicalRestore(_ context.Context, req *appservices.PostgresLogicalRestoreRequest, reader io.Reader) error {
	c.restoreReq = req
	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	c.restoreContents = body
	return c.restoreErr
}

func TestPostgresRunnerBackupStrategyExecute(t *testing.T) {
	client := &fakePostgresBackupRunnerClient{
		dumpArtifact: &appservices.BackupRunnerArtifact{
			FileName: "app.dump.gz",
			Metadata: map[string]any{"postgres_version": "17", "tool": "pg_dump", "dump_format": "custom"},
		},
	}
	strategy := NewPostgresRunnerBackupStrategy(client)
	source := &domain.DatabaseSource{
		DBType:            domain.DatabaseTypePostgres,
		Version:           "17",
		Host:              "127.0.0.1",
		Port:              5432,
		Username:          "postgres",
		PasswordEncrypted: "secret",
		DatabaseName:      "app",
	}
	config := &domain.BackupConfig{
		CompressionType: domain.BackupCompressionGzip,
		BackupMethod:    domain.BackupMethodLogicalDump,
	}

	artifact, err := strategy.Execute(context.Background(), source, config)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	defer os.Remove(artifact.LocalPath)
	if artifact.FileName != "app.dump.gz" {
		t.Fatalf("file name = %s", artifact.FileName)
	}
	if artifact.Metadata["tool"] != "pg_dump" {
		t.Fatalf("tool = %v", artifact.Metadata["tool"])
	}
	data, err := os.ReadFile(artifact.LocalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "postgres-dump-data" {
		t.Fatalf("artifact contents = %q", string(data))
	}
}

func TestPostgresRunnerRestoreStrategyExecute(t *testing.T) {
	client := &fakePostgresBackupRunnerClient{}
	strategy := NewPostgresRunnerRestoreStrategy(client)
	localPath := filepath.Join(t.TempDir(), "app.dump.gz")
	if err := os.WriteFile(localPath, []byte("restore-data"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := strategy.Execute(context.Background(), &domain.DatabaseSource{
		Version:           "17",
		Host:              "127.0.0.1",
		Port:              5432,
		Username:          "postgres",
		PasswordEncrypted: "secret",
		DatabaseName:      "app",
	}, &domain.Backup{
		Metadata: map[string]any{"compression_type": "gzip"},
	}, localPath)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.restoreReq == nil {
		t.Fatal("expected restore request")
	}
	if client.restoreReq.CompressionType != domain.BackupCompressionGzip {
		t.Fatalf("compression = %s", client.restoreReq.CompressionType)
	}
	if string(client.restoreContents) != "restore-data" {
		t.Fatalf("restore contents = %q", string(client.restoreContents))
	}
}
