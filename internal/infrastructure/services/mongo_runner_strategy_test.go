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

type fakeMongoBackupRunnerClient struct {
	dumpArtifact    *appservices.BackupRunnerArtifact
	dumpErr         error
	restoreErr      error
	restoreReq      *appservices.MongoLogicalRestoreRequest
	restoreContents []byte
}

func (c *fakeMongoBackupRunnerClient) RunMySQLLogicalDump(_ context.Context, _ *appservices.MySQLLogicalDumpRequest, _ io.Writer) (*appservices.BackupRunnerArtifact, error) {
	return nil, nil
}

func (c *fakeMongoBackupRunnerClient) RunMySQLLogicalRestore(_ context.Context, _ *appservices.MySQLLogicalRestoreRequest, _ io.Reader) error {
	return nil
}

func (c *fakeMongoBackupRunnerClient) RunPostgresLogicalDump(_ context.Context, _ *appservices.PostgresLogicalDumpRequest, _ io.Writer) (*appservices.BackupRunnerArtifact, error) {
	return nil, nil
}

func (c *fakeMongoBackupRunnerClient) RunPostgresLogicalRestore(_ context.Context, _ *appservices.PostgresLogicalRestoreRequest, _ io.Reader) error {
	return nil
}

func (c *fakeMongoBackupRunnerClient) RunMongoLogicalDump(_ context.Context, _ *appservices.MongoLogicalDumpRequest, writer io.Writer) (*appservices.BackupRunnerArtifact, error) {
	if c.dumpErr != nil {
		return nil, c.dumpErr
	}
	if _, err := writer.Write([]byte("mongo-dump-data")); err != nil {
		return nil, err
	}
	return c.dumpArtifact, nil
}

func (c *fakeMongoBackupRunnerClient) RunMongoLogicalRestore(_ context.Context, req *appservices.MongoLogicalRestoreRequest, reader io.Reader) error {
	c.restoreReq = req
	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	c.restoreContents = body
	return c.restoreErr
}

func TestMongoRunnerBackupStrategyExecute(t *testing.T) {
	client := &fakeMongoBackupRunnerClient{
		dumpArtifact: &appservices.BackupRunnerArtifact{
			FileName: "app.archive.gz",
			Metadata: map[string]any{"database_name": "app", "tool": "mongodump"},
		},
	}
	strategy := NewMongoRunnerBackupStrategy(client)
	source := &domain.DatabaseSource{
		DBType:                 domain.DatabaseTypeMongoDB,
		Host:                   "127.0.0.1",
		Port:                   27017,
		Username:               "root",
		PasswordEncrypted:      "secret",
		DatabaseName:           "app",
		AuthDatabase:           "admin",
		ConnectionURIEncrypted: "mongodb://root:secret@127.0.0.1:27017/?authSource=admin",
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
	if artifact.FileName != "app.archive.gz" {
		t.Fatalf("file name = %s", artifact.FileName)
	}
	if artifact.Metadata["tool"] != "mongodump" {
		t.Fatalf("tool = %v", artifact.Metadata["tool"])
	}
	data, err := os.ReadFile(artifact.LocalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mongo-dump-data" {
		t.Fatalf("artifact contents = %q", string(data))
	}
}

func TestMongoRunnerRestoreStrategyExecute(t *testing.T) {
	client := &fakeMongoBackupRunnerClient{}
	strategy := NewMongoRunnerRestoreStrategy(client)
	localPath := filepath.Join(t.TempDir(), "app.archive.gz")
	if err := os.WriteFile(localPath, []byte("restore-data"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := strategy.Execute(context.Background(), &domain.DatabaseSource{
		Host:                   "127.0.0.1",
		Port:                   27017,
		Username:               "root",
		PasswordEncrypted:      "secret",
		DatabaseName:           "app_restore",
		AuthDatabase:           "admin",
		ConnectionURIEncrypted: "mongodb://root:secret@127.0.0.1:27017/?authSource=admin",
	}, &domain.Backup{
		Metadata: map[string]any{
			"compression_type": "gzip",
			"database_name":    "app",
		},
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
	if client.restoreReq.SourceDatabase != "app" {
		t.Fatalf("source database = %s", client.restoreReq.SourceDatabase)
	}
	if client.restoreReq.Database != "app_restore" {
		t.Fatalf("target database = %s", client.restoreReq.Database)
	}
	if string(client.restoreContents) != "restore-data" {
		t.Fatalf("restore contents = %q", string(client.restoreContents))
	}
}
