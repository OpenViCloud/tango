package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeMySQLVersion(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{raw: "9", want: "9"},
		{raw: "8.4", want: "8.4"},
		{raw: "8.0.36", want: "8.0"},
		{raw: "8.4.1", want: "8.4"},
		{raw: "9.2.0", want: "9"},
		{raw: "5.7.44-log", want: "5.7"},
		{raw: "10.1.0", wantErr: true},
	}
	for _, tt := range tests {
		got, err := NormalizeMySQLVersion(tt.raw)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("NormalizeMySQLVersion(%q) expected error", tt.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizeMySQLVersion(%q) error = %v", tt.raw, err)
		}
		if got != tt.want {
			t.Fatalf("NormalizeMySQLVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestGetMySQLExecutable(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "mysql-8.4", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(binDir, "mysqldump")
	if err := os.WriteFile(target, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := GetMySQLExecutable("8.4.1", MySQLExecutableDump, root)
	if err != nil {
		t.Fatalf("GetMySQLExecutable() error = %v", err)
	}
	if got != target {
		t.Fatalf("GetMySQLExecutable() = %q, want %q", got, target)
	}
}

func TestVerifyMySQLInstallation(t *testing.T) {
	root := t.TempDir()
	for _, version := range []string{"5.7", "8.0", "8.4", "9"} {
		binDir := filepath.Join(root, "mysql-"+version, "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for _, executable := range []string{"mysql", "mysqldump"} {
			if err := os.WriteFile(filepath.Join(binDir, executable), []byte(""), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := VerifyMySQLInstallation(root); err != nil {
		t.Fatalf("VerifyMySQLInstallation() error = %v", err)
	}
}

func TestVerifyMySQLInstallationMissingBinary(t *testing.T) {
	root := t.TempDir()
	for _, version := range []string{"5.7", "8.0", "8.4", "9"} {
		binDir := filepath.Join(root, "mysql-"+version, "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(binDir, "mysql"), []byte(""), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	err := VerifyMySQLInstallation(root)
	if err == nil {
		t.Fatal("VerifyMySQLInstallation() expected error")
	}
}
