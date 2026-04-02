package services

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"tango/internal/domain"
)

type fakeMariaDBDumpProcessFactory struct {
	stdout  string
	stderr  string
	waitErr error
}

type fakeMariaDBDumpProcess struct {
	stdout  string
	stderr  string
	waitErr error
}

type fakeMariaDBBinaryResolver struct {
	dumpPath  string
	clientPath string
	err       error
}

func (r *fakeMariaDBBinaryResolver) MariaDBDump(_ string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return r.dumpPath, nil
}

func (r *fakeMariaDBBinaryResolver) MariaDB(_ string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return r.clientPath, nil
}

func (f *fakeMariaDBDumpProcessFactory) New(_ context.Context, _ string, _ ...string) mariaDBDumpProcess {
	return &fakeMariaDBDumpProcess{stdout: f.stdout, stderr: f.stderr, waitErr: f.waitErr}
}

func (p *fakeMariaDBDumpProcess) StdoutPipe() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(p.stdout)), nil
}
func (p *fakeMariaDBDumpProcess) StderrPipe() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(p.stderr)), nil
}
func (p *fakeMariaDBDumpProcess) Start() error { return nil }
func (p *fakeMariaDBDumpProcess) Wait() error  { return p.waitErr }

func TestMariaDBLogicalDumpStrategyExecuteGzip(t *testing.T) {
	strategy := newMariaDBLogicalDumpStrategyWithFactory(&fakeMariaDBDumpProcessFactory{
		stdout: "CREATE TABLE test (id int);\nINSERT INTO test VALUES (1);\n",
	}, &fakeMariaDBBinaryResolver{dumpPath: "/usr/local/mariadb-11.4/bin/mariadb-dump"})
	source := &domain.DatabaseSource{
		ID:                "src_1",
		DBType:            domain.DatabaseTypeMariaDB,
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "user",
		PasswordEncrypted: "secret",
		DatabaseName:      "app_db",
		Version:           "11.4",
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

	file, err := os.Open(artifact.LocalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "CREATE TABLE test") {
		t.Fatalf("unexpected dump content: %s", string(raw))
	}
	if artifact.FileName != "app_db.sql.gz" {
		t.Fatalf("file name = %s", artifact.FileName)
	}
}

func TestMariaDBLogicalDumpStrategyExecuteFailure(t *testing.T) {
	strategy := newMariaDBLogicalDumpStrategyWithFactory(&fakeMariaDBDumpProcessFactory{
		stderr:  "permission denied",
		waitErr: errors.New("exit status 2"),
	}, &fakeMariaDBBinaryResolver{dumpPath: "/usr/local/mariadb-11.4/bin/mariadb-dump"})
	source := &domain.DatabaseSource{
		ID:                "src_1",
		DBType:            domain.DatabaseTypeMariaDB,
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "user",
		PasswordEncrypted: "secret",
		DatabaseName:      "app_db",
		Version:           "11.4",
	}
	config := &domain.BackupConfig{
		CompressionType: domain.BackupCompressionGzip,
		BackupMethod:    domain.BackupMethodLogicalDump,
	}

	if _, err := strategy.Execute(context.Background(), source, config); err == nil {
		t.Fatal("expected error")
	}
}
