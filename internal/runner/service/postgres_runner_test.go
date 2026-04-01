package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tango/internal/runner/model"
)

func TestPostgresRunnerRunLogicalDump(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "17", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dumpScript := filepath.Join(binDir, "pg_dump")
	if err := os.WriteFile(dumpScript, []byte(`#!/bin/sh
printf 'dump-data'
`), 0o755); err != nil {
		t.Fatal(err)
	}

	runner := NewPostgresRunner(root)
	var output strings.Builder
	fileName, err := runner.RunLogicalDump(context.Background(), &model.PostgresLogicalDumpRequest{
		Version:         "17",
		Host:            "127.0.0.1",
		Port:            5432,
		Username:        "postgres",
		Password:        "secret",
		Database:        "app",
		CompressionType: "none",
	}, &output)
	if err != nil {
		t.Fatalf("RunLogicalDump() error = %v", err)
	}
	if fileName != "app.dump" {
		t.Fatalf("file name = %s", fileName)
	}
	if output.String() != "dump-data" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestPostgresRunnerRunLogicalRestore(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "17", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	restoreScript := filepath.Join(binDir, "pg_restore")
	captureFile := filepath.Join(t.TempDir(), "restore.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	script := "#!/bin/sh\n" +
		"cat > " + captureFile + "\n" +
		"printf '%s' \"$*\" > " + argsFile + "\n"
	if err := os.WriteFile(restoreScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	runner := NewPostgresRunner(root)
	err := runner.RunLogicalRestore(context.Background(), &model.PostgresLogicalDumpRequest{
		Version:         "17",
		Host:            "127.0.0.1",
		Port:            5432,
		Username:        "postgres",
		Password:        "secret",
		Database:        "app",
		CompressionType: "none",
	}, strings.NewReader("dump-data"))
	if err != nil {
		t.Fatalf("RunLogicalRestore() error = %v", err)
	}
	body, err := os.ReadFile(captureFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "dump-data" {
		t.Fatalf("restore body = %q", string(body))
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "--clean") {
		t.Fatalf("args missing --clean: %s", string(args))
	}
}
