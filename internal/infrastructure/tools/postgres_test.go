package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePostgresVersion(t *testing.T) {
	tests := map[string]string{
		"12":     "12",
		"13.18":  "13",
		"14.12":  "14",
		"15.7":   "15",
		"16.4":   "16",
		"17.5":   "17",
		"18beta": "18",
	}
	for raw, expected := range tests {
		got, err := NormalizePostgresVersion(raw)
		if err != nil {
			t.Fatalf("NormalizePostgresVersion(%q) error = %v", raw, err)
		}
		if got != expected {
			t.Fatalf("NormalizePostgresVersion(%q) = %q, want %q", raw, got, expected)
		}
	}
}

func TestGetPostgresExecutable(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "17", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(binDir, "pg_dump")
	if err := os.WriteFile(path, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, err := GetPostgresExecutable("17.5", PostgresExecutableDump, root)
	if err != nil {
		t.Fatalf("GetPostgresExecutable() error = %v", err)
	}
	if resolved != path {
		t.Fatalf("path = %s, want %s", resolved, path)
	}
}
