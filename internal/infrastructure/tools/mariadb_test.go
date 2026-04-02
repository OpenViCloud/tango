package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeMariaDBVersion(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{raw: "12.1", want: "12.1"},
		{raw: "12.1.2-MariaDB-ubu2404", want: "12.1"},
		{raw: "10.6.21-MariaDB", want: "10.6"},
		{raw: "11.4.1-MariaDB", wantErr: true},
	}
	for _, tt := range tests {
		got, err := NormalizeMariaDBVersion(tt.raw)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("NormalizeMariaDBVersion(%q) expected error", tt.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizeMariaDBVersion(%q) error = %v", tt.raw, err)
		}
		if got != tt.want {
			t.Fatalf("NormalizeMariaDBVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestGetMariaDBExecutable(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "mariadb-12.1", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(binDir, "mariadb-dump")
	if err := os.WriteFile(target, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := GetMariaDBExecutable("12.1.2-MariaDB-ubu2404", MariaDBExecutableDump, root)
	if err != nil {
		t.Fatalf("GetMariaDBExecutable() error = %v", err)
	}
	if got != target {
		t.Fatalf("GetMariaDBExecutable() = %q, want %q", got, target)
	}
}

func TestVerifyMariaDBInstallation(t *testing.T) {
	root := t.TempDir()
	for _, version := range []string{"10.6", "12.1"} {
		binDir := filepath.Join(root, "mariadb-"+version, "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for _, executable := range []string{"mariadb", "mariadb-dump"} {
			if err := os.WriteFile(filepath.Join(binDir, executable), []byte(""), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := VerifyMariaDBInstallation(root); err != nil {
		t.Fatalf("VerifyMariaDBInstallation() error = %v", err)
	}
}

func TestVerifyMariaDBInstallationMissingBinary(t *testing.T) {
	root := t.TempDir()
	for _, version := range []string{"10.6", "12.1"} {
		binDir := filepath.Join(root, "mariadb-"+version, "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(binDir, "mariadb"), []byte(""), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	err := VerifyMariaDBInstallation(root)
	if err == nil {
		t.Fatal("VerifyMariaDBInstallation() expected error")
	}
}
