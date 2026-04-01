package services

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tango/internal/domain"
)

type fakeRestoreProcessFactory struct {
	waitErr error
	stderr  string
	written strings.Builder
}

type fakeRestoreProcess struct {
	waitErr error
	stderr  string
	writer  *captureWriteCloser
}

type captureWriteCloser struct {
	builder *strings.Builder
}

func (w *captureWriteCloser) Write(p []byte) (int, error) {
	return w.builder.Write(p)
}
func (w *captureWriteCloser) Close() error { return nil }

func (f *fakeRestoreProcessFactory) New(_ context.Context, _ string, _ ...string) restoreProcess {
	return &fakeRestoreProcess{
		waitErr: f.waitErr,
		stderr:  f.stderr,
		writer:  &captureWriteCloser{builder: &f.written},
	}
}

func (p *fakeRestoreProcess) StdinPipe() (io.WriteCloser, error) { return p.writer, nil }
func (p *fakeRestoreProcess) StderrPipe() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(p.stderr)), nil
}
func (p *fakeRestoreProcess) Start() error { return nil }
func (p *fakeRestoreProcess) Wait() error  { return p.waitErr }

func TestMySQLRestoreStrategyExecuteGzip(t *testing.T) {
	factory := &fakeRestoreProcessFactory{}
	strategy := newMySQLRestoreStrategyWithFactory(factory, &fakeMySQLBinaryResolver{mysqlPath: "/usr/local/mysql-8.4/bin/mysql"})
	target := &domain.DatabaseSource{
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "user",
		PasswordEncrypted: "secret",
		DatabaseName:      "app_db",
		Version:           "8.4",
	}
	backup := &domain.Backup{BackupMethod: domain.BackupMethodLogicalDump}
	localPath := filepath.Join(t.TempDir(), "app.sql.gz")
	if err := writeGzipFile(localPath, "CREATE TABLE test (id int);\n"); err != nil {
		t.Fatal(err)
	}
	if err := strategy.Execute(context.Background(), target, backup, localPath); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(factory.written.String(), "CREATE TABLE test") {
		t.Fatalf("unexpected restore input: %s", factory.written.String())
	}
}

func TestMySQLRestoreStrategyExecuteFailure(t *testing.T) {
	factory := &fakeRestoreProcessFactory{waitErr: errors.New("exit status 1"), stderr: "syntax error"}
	strategy := newMySQLRestoreStrategyWithFactory(factory, &fakeMySQLBinaryResolver{mysqlPath: "/usr/local/mysql-8.4/bin/mysql"})
	target := &domain.DatabaseSource{
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "user",
		PasswordEncrypted: "secret",
		DatabaseName:      "app_db",
		Version:           "8.4",
	}
	backup := &domain.Backup{BackupMethod: domain.BackupMethodLogicalDump}
	localPath := filepath.Join(t.TempDir(), "app.sql")
	if err := os.WriteFile(localPath, []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := strategy.Execute(context.Background(), target, backup, localPath); err == nil {
		t.Fatal("expected error")
	}
}

func writeGzipFile(path string, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := gzip.NewWriter(f)
	if _, err := w.Write([]byte(content)); err != nil {
		return err
	}
	return w.Close()
}
