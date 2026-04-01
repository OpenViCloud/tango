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

type fakeDumpProcessFactory struct {
	stdout  string
	stderr  string
	waitErr error
}

type fakeDumpProcess struct {
	stdout  string
	stderr  string
	waitErr error
}

type fakeMySQLBinaryResolver struct {
	dumpPath  string
	mysqlPath string
	err       error
}

func (r *fakeMySQLBinaryResolver) Mysqldump(_ string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return r.dumpPath, nil
}

func (r *fakeMySQLBinaryResolver) Mysql(_ string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return r.mysqlPath, nil
}

func (f *fakeDumpProcessFactory) New(_ context.Context, _ string, _ ...string) dumpProcess {
	return &fakeDumpProcess{stdout: f.stdout, stderr: f.stderr, waitErr: f.waitErr}
}

func (p *fakeDumpProcess) StdoutPipe() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(p.stdout)), nil
}
func (p *fakeDumpProcess) StderrPipe() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(p.stderr)), nil
}
func (p *fakeDumpProcess) Start() error { return nil }
func (p *fakeDumpProcess) Wait() error  { return p.waitErr }

func TestMySQLLogicalDumpStrategyExecuteGzip(t *testing.T) {
	strategy := newMySQLLogicalDumpStrategyWithFactory(&fakeDumpProcessFactory{
		stdout: "CREATE TABLE test (id int);\nINSERT INTO test VALUES (1);\n",
	}, &fakeMySQLBinaryResolver{dumpPath: "/usr/local/mysql-8.4/bin/mysqldump"})
	source := &domain.DatabaseSource{
		ID:                "src_1",
		DBType:            domain.DatabaseTypeMySQL,
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "user",
		PasswordEncrypted: "secret",
		DatabaseName:      "app_db",
		Version:           "8.4",
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

func TestMySQLLogicalDumpStrategyExecuteFailure(t *testing.T) {
	strategy := newMySQLLogicalDumpStrategyWithFactory(&fakeDumpProcessFactory{
		stderr:  "permission denied",
		waitErr: errors.New("exit status 2"),
	}, &fakeMySQLBinaryResolver{dumpPath: "/usr/local/mysql-8.4/bin/mysqldump"})
	source := &domain.DatabaseSource{
		ID:                "src_1",
		DBType:            domain.DatabaseTypeMySQL,
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "user",
		PasswordEncrypted: "secret",
		DatabaseName:      "app_db",
		Version:           "8.4",
	}
	config := &domain.BackupConfig{
		CompressionType: domain.BackupCompressionGzip,
		BackupMethod:    domain.BackupMethodLogicalDump,
	}

	if _, err := strategy.Execute(context.Background(), source, config); err == nil {
		t.Fatal("expected error")
	}
}
