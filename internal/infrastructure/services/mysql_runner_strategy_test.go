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

type fakeBackupRunnerClient struct {
	dumpArtifact    *appservices.BackupRunnerArtifact
	dumpErr         error
	restoreErr      error
	restoreReq      *appservices.MySQLLogicalRestoreRequest
	restoreContents []byte
}

func (c *fakeBackupRunnerClient) RunMySQLLogicalDump(_ context.Context, _ *appservices.MySQLLogicalDumpRequest, writer io.Writer) (*appservices.BackupRunnerArtifact, error) {
	if c.dumpErr != nil {
		return nil, c.dumpErr
	}
	if _, err := writer.Write([]byte("dump-data")); err != nil {
		return nil, err
	}
	return c.dumpArtifact, nil
}

func (c *fakeBackupRunnerClient) RunMySQLLogicalRestore(_ context.Context, req *appservices.MySQLLogicalRestoreRequest, reader io.Reader) error {
	c.restoreReq = req
	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	c.restoreContents = body
	return c.restoreErr
}

func (c *fakeBackupRunnerClient) RunPostgresLogicalDump(_ context.Context, _ *appservices.PostgresLogicalDumpRequest, _ io.Writer) (*appservices.BackupRunnerArtifact, error) {
	return nil, nil
}

func (c *fakeBackupRunnerClient) RunPostgresLogicalRestore(_ context.Context, _ *appservices.PostgresLogicalRestoreRequest, _ io.Reader) error {
	return nil
}

func (c *fakeBackupRunnerClient) RunMongoLogicalDump(_ context.Context, _ *appservices.MongoLogicalDumpRequest, _ io.Writer) (*appservices.BackupRunnerArtifact, error) {
	return nil, nil
}

func (c *fakeBackupRunnerClient) RunMongoLogicalRestore(_ context.Context, _ *appservices.MongoLogicalRestoreRequest, _ io.Reader) error {
	return nil
}

func TestMySQLRunnerBackupStrategyExecute(t *testing.T) {
	client := &fakeBackupRunnerClient{
		dumpArtifact: &appservices.BackupRunnerArtifact{
			FileName: "app.sql.gz",
			Metadata: map[string]any{"mysql_version": "9", "tool": "mysqldump"},
		},
	}
	strategy := NewMySQLRunnerBackupStrategy(client)
	source := &domain.DatabaseSource{
		DBType:            domain.DatabaseTypeMySQL,
		Version:           "9",
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
	if artifact.Metadata["tool"] != "mysqldump" {
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

func TestMySQLRunnerRestoreStrategyExecute(t *testing.T) {
	client := &fakeBackupRunnerClient{}
	strategy := NewMySQLRunnerRestoreStrategy(client)
	localPath := filepath.Join(t.TempDir(), "app.sql.gz")
	if err := os.WriteFile(localPath, []byte("restore-data"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := strategy.Execute(context.Background(), &domain.DatabaseSource{
		Version:           "9",
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
