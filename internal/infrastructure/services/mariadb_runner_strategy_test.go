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

type fakeMariaDBBackupRunnerClient struct {
	dumpArtifact    *appservices.BackupRunnerArtifact
	dumpErr         error
	restoreErr      error
	restoreReq      *appservices.MariaDBLogicalRestoreRequest
	restoreContents []byte
}

func (c *fakeMariaDBBackupRunnerClient) RunMySQLLogicalDump(_ context.Context, _ *appservices.MySQLLogicalDumpRequest, _ io.Writer) (*appservices.BackupRunnerArtifact, error) {
	return nil, nil
}

func (c *fakeMariaDBBackupRunnerClient) RunMySQLLogicalRestore(_ context.Context, _ *appservices.MySQLLogicalRestoreRequest, _ io.Reader) error {
	return nil
}

func (c *fakeMariaDBBackupRunnerClient) RunMariaDBLogicalDump(_ context.Context, _ *appservices.MariaDBLogicalDumpRequest, writer io.Writer) (*appservices.BackupRunnerArtifact, error) {
	if c.dumpErr != nil {
		return nil, c.dumpErr
	}
	if _, err := writer.Write([]byte("dump-data")); err != nil {
		return nil, err
	}
	return c.dumpArtifact, nil
}

func (c *fakeMariaDBBackupRunnerClient) RunMariaDBLogicalRestore(_ context.Context, req *appservices.MariaDBLogicalRestoreRequest, reader io.Reader) error {
	c.restoreReq = req
	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	c.restoreContents = body
	return c.restoreErr
}

func (c *fakeMariaDBBackupRunnerClient) RunPostgresLogicalDump(_ context.Context, _ *appservices.PostgresLogicalDumpRequest, _ io.Writer) (*appservices.BackupRunnerArtifact, error) {
	return nil, nil
}

func (c *fakeMariaDBBackupRunnerClient) RunPostgresLogicalRestore(_ context.Context, _ *appservices.PostgresLogicalRestoreRequest, _ io.Reader) error {
	return nil
}

func (c *fakeMariaDBBackupRunnerClient) RunMongoLogicalDump(_ context.Context, _ *appservices.MongoLogicalDumpRequest, _ io.Writer) (*appservices.BackupRunnerArtifact, error) {
	return nil, nil
}

func (c *fakeMariaDBBackupRunnerClient) RunMongoLogicalRestore(_ context.Context, _ *appservices.MongoLogicalRestoreRequest, _ io.Reader) error {
	return nil
}

func TestMariaDBRunnerBackupStrategyExecute(t *testing.T) {
	client := &fakeMariaDBBackupRunnerClient{
		dumpArtifact: &appservices.BackupRunnerArtifact{
			FileName: "app.sql.gz",
			Metadata: map[string]any{"mariadb_version": "11.4", "tool": "mariadb-dump"},
		},
	}
	strategy := NewMariaDBRunnerBackupStrategy(client)
	source := &domain.DatabaseSource{
		DBType:            domain.DatabaseTypeMariaDB,
		Version:           "11.4",
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "root",
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
	if artifact.FileName != "app.sql.gz" {
		t.Fatalf("file name = %s", artifact.FileName)
	}
	if artifact.Metadata["tool"] != "mariadb-dump" {
		t.Fatalf("tool = %v", artifact.Metadata["tool"])
	}
	data, err := os.ReadFile(artifact.LocalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "dump-data" {
		t.Fatalf("artifact contents = %q", string(data))
	}
}

func TestMariaDBRunnerRestoreStrategyExecute(t *testing.T) {
	client := &fakeMariaDBBackupRunnerClient{}
	strategy := NewMariaDBRunnerRestoreStrategy(client)
	localPath := filepath.Join(t.TempDir(), "app.sql.gz")
	if err := os.WriteFile(localPath, []byte("restore-data"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := strategy.Execute(context.Background(), &domain.DatabaseSource{
		Version:           "11.4",
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "root",
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
