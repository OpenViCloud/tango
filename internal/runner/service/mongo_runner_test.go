package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tango/internal/runner/model"
)

func TestMongoRunnerRunLogicalDump(t *testing.T) {
	toolsDir := filepath.Join(t.TempDir(), "mongodb-database-tools")
	binDir := filepath.Join(toolsDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dumpScript := filepath.Join(binDir, "mongodump")
	if err := os.WriteFile(dumpScript, []byte(`#!/bin/sh
printf 'archive-data'
`), 0o755); err != nil {
		t.Fatal(err)
	}

	runner := NewMongoRunner(toolsDir)
	var output strings.Builder
	fileName, err := runner.RunLogicalDump(context.Background(), &model.MongoLogicalDumpRequest{
		Host:            "127.0.0.1",
		Port:            27017,
		Username:        "root",
		Password:        "secret",
		Database:        "app",
		AuthDatabase:    "admin",
		CompressionType: "gzip",
	}, &output)
	if err != nil {
		t.Fatalf("RunLogicalDump() error = %v", err)
	}
	if fileName != "app.archive.gz" {
		t.Fatalf("file name = %s", fileName)
	}
	if output.String() != "archive-data" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestMongoRunnerRunLogicalRestore(t *testing.T) {
	toolsDir := filepath.Join(t.TempDir(), "mongodb-database-tools")
	binDir := filepath.Join(toolsDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	restoreScript := filepath.Join(binDir, "mongorestore")
	captureFile := filepath.Join(t.TempDir(), "restore.txt")
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	script := "#!/bin/sh\n" +
		"cat > " + captureFile + "\n" +
		"printf '%s' \"$*\" > " + argsFile + "\n"
	if err := os.WriteFile(restoreScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	runner := NewMongoRunner(toolsDir)
	err := runner.RunLogicalRestore(context.Background(), &model.MongoLogicalRestoreRequest{
		Host:            "127.0.0.1",
		Port:            27017,
		Username:        "root",
		Password:        "secret",
		Database:        "app_restore",
		AuthDatabase:    "admin",
		SourceDatabase:  "app",
		CompressionType: "gzip",
	}, strings.NewReader("archive-data"))
	if err != nil {
		t.Fatalf("RunLogicalRestore() error = %v", err)
	}

	body, err := os.ReadFile(captureFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "archive-data" {
		t.Fatalf("restore body = %q", string(body))
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "--nsFrom=app.*") {
		t.Fatalf("args missing nsFrom remap: %s", string(args))
	}
	if !strings.Contains(string(args), "--nsTo=app_restore.*") {
		t.Fatalf("args missing nsTo remap: %s", string(args))
	}
}
