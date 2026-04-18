package command

import (
	"testing"

	"tango/internal/domain"
)

func TestApplyRandomVolumePrefix(t *testing.T) {
	orig := newRandomVolumePrefix
	newRandomVolumePrefix = func() (string, error) { return "a1b2", nil }
	t.Cleanup(func() { newRandomVolumePrefix = orig })

	volumes, volumeFiles, err := ApplyRandomVolumePrefix(
		[]string{
			"n8n-postgres-data:/var/lib/postgresql/data",
			"shared/cache:/cache:ro",
		},
		[]domain.VolumeFileTemplate{
			{Path: "openclaw-config/openclaw.json", Content: "{}"},
		},
	)
	if err != nil {
		t.Fatalf("ApplyRandomVolumePrefix() error = %v", err)
	}

	if got, want := volumes[0], "a1b2-n8n-postgres-data:/var/lib/postgresql/data"; got != want {
		t.Fatalf("volumes[0] = %q, want %q", got, want)
	}
	if got, want := volumes[1], "a1b2-shared/cache:/cache:ro"; got != want {
		t.Fatalf("volumes[1] = %q, want %q", got, want)
	}
	if got, want := volumeFiles[0].Path, "a1b2-openclaw-config/openclaw.json"; got != want {
		t.Fatalf("volumeFiles[0].Path = %q, want %q", got, want)
	}
}

func TestApplyRandomVolumePrefix_NoStorage(t *testing.T) {
	volumes, volumeFiles, err := ApplyRandomVolumePrefix(nil, nil)
	if err != nil {
		t.Fatalf("ApplyRandomVolumePrefix() error = %v", err)
	}
	if volumes != nil {
		t.Fatalf("volumes = %#v, want nil", volumes)
	}
	if volumeFiles != nil {
		t.Fatalf("volumeFiles = %#v, want nil", volumeFiles)
	}
}
