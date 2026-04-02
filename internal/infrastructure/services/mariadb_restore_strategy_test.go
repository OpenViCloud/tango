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

type fakeMariaDBRestoreProcessFactory struct {
	waitErr error
	stderr  string
	written strings.Builder
}

type fakeMariaDBRestoreProcess struct {
	waitErr error
	stderr  string
	writer  *mariaDBCaptureWriteCloser
}

type mariaDBCaptureWriteCloser struct {
	builder *strings.Builder
}

func (w *mariaDBCaptureWriteCloser) Write(p []byte) (int, error) {
	return w.builder.Write(p)
}
func (w *mariaDBCaptureWriteCloser) Close() error { return nil }

func (f *fakeMariaDBRestoreProcessFactory) New(_ context.Context, _ string, _ ...string) mariaDBRestoreProcess {
	return &fakeMariaDBRestoreProcess{
		waitErr: f.waitErr,
		stderr:  f.stderr,
		writer:  &mariaDBCaptureWriteCloser{builder: &f.written},
	}
}

func (p *fakeMariaDBRestoreProcess) StdinPipe() (io.WriteCloser, error) { return p.writer, nil }
func (p *fakeMariaDBRestoreProcess) StderrPipe() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(p.stderr)), nil
}
func (p *fakeMariaDBRestoreProcess) Start() error { return nil }
func (p *fakeMariaDBRestoreProcess) Wait() error  { return p.waitErr }

func TestMariaDBRestoreStrategyExecuteGzip(t *testing.T) {
	factory := &fakeMariaDBRestoreProcessFactory{}
	strategy := newMariaDBRestoreStrategyWithFactory(factory, &fakeMariaDBBinaryResolver{clientPath: "/usr/local/mariadb-11.4/bin/mariadb"})
	target := &domain.DatabaseSource{
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "user",
		PasswordEncrypted: "secret",
		DatabaseName:      "app_db",
		Version:           "11.4",
	}
	backup := &domain.Backup{BackupMethod: domain.BackupMethodLogicalDump}
	localPath := filepath.Join(t.TempDir(), "app.sql.gz")
	if err := writeMariaDBGzipFile(localPath, "CREATE TABLE test (id int);\n"); err != nil {
		t.Fatal(err)
	}
	if err := strategy.Execute(context.Background(), target, backup, localPath); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(factory.written.String(), "CREATE TABLE test") {
		t.Fatalf("unexpected restore input: %s", factory.written.String())
	}
}

func TestMariaDBRestoreStrategyExecuteFailure(t *testing.T) {
	factory := &fakeMariaDBRestoreProcessFactory{waitErr: errors.New("exit status 1"), stderr: "syntax error"}
	strategy := newMariaDBRestoreStrategyWithFactory(factory, &fakeMariaDBBinaryResolver{clientPath: "/usr/local/mariadb-11.4/bin/mariadb"})
	target := &domain.DatabaseSource{
		Host:              "127.0.0.1",
		Port:              3306,
		Username:          "user",
		PasswordEncrypted: "secret",
		DatabaseName:      "app_db",
		Version:           "11.4",
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

func writeMariaDBGzipFile(path string, content string) error {
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
