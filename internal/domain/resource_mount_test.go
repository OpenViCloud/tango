package domain

import "testing"

func TestResolveResourceMounts(t *testing.T) {
	t.Parallel()

	cfg := map[string]any{
		"volumes": []interface{}{
			"databasus-data:/databasus-data",
			"shared/cache:/data:ro",
		},
	}

	mounts, err := ResolveResourceMounts(cfg, "/tmp/tango-resource-volumes")
	if err != nil {
		t.Fatalf("ResolveResourceMounts() error = %v", err)
	}

	if len(mounts.Binds) != 2 {
		t.Fatalf("len(Binds) = %d, want 2", len(mounts.Binds))
	}
	if mounts.Binds[0] != "/tmp/tango-resource-volumes/databasus-data:/databasus-data" {
		t.Fatalf("first bind = %q", mounts.Binds[0])
	}
	if mounts.Binds[1] != "/tmp/tango-resource-volumes/shared/cache:/data:ro" {
		t.Fatalf("second bind = %q", mounts.Binds[1])
	}
}

func TestResolveResourceMountsRejectsTraversal(t *testing.T) {
	t.Parallel()

	cfg := map[string]any{
		"volumes": []interface{}{"../escape:/data"},
	}

	if _, err := ResolveResourceMounts(cfg, "/tmp/tango-resource-volumes"); err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestResolveResourceMountsRejectsRelativeTarget(t *testing.T) {
	t.Parallel()

	cfg := map[string]any{
		"volumes": []interface{}{"data:relative/path"},
	}

	if _, err := ResolveResourceMounts(cfg, "/tmp/tango-resource-volumes"); err == nil {
		t.Fatal("expected invalid target error")
	}
}
