package domain

import (
	"path"
	"path/filepath"
	"strings"
)

type ResourceMounts struct {
	Binds     []string
	HostPaths []string
}

func ResolveResourceMounts(cfg map[string]any, mountRoot string) (ResourceMounts, error) {
	if cfg == nil {
		return ResourceMounts{}, nil
	}

	raw, ok := cfg["volumes"]
	if !ok {
		return ResourceMounts{}, nil
	}

	items, ok := raw.([]interface{})
	if !ok {
		return ResourceMounts{}, NewUserFacingError("Resource volumes must be a list of mount definitions")
	}

	root := filepath.Clean(strings.TrimSpace(mountRoot))
	if root == "" || root == "." {
		return ResourceMounts{}, NewUserFacingError("Resource mount root is not configured")
	}

	result := ResourceMounts{
		Binds:     make([]string, 0, len(items)),
		HostPaths: make([]string, 0, len(items)),
	}
	for _, item := range items {
		mount, ok := item.(string)
		if !ok {
			return ResourceMounts{}, NewUserFacingError("Resource volumes must be strings in source:target[:mode] format")
		}
		bind, hostPath, err := resolveResourceMount(strings.TrimSpace(mount), root)
		if err != nil {
			return ResourceMounts{}, err
		}
		result.Binds = append(result.Binds, bind)
		result.HostPaths = append(result.HostPaths, hostPath)
	}

	return result, nil
}

func resolveResourceMount(input, root string) (string, string, error) {
	parts := strings.Split(input, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return "", "", NewUserFacingError("Invalid volume mount format. Use source:target[:mode]")
	}

	source := strings.TrimSpace(parts[0])
	target := strings.TrimSpace(parts[1])
	mode := ""
	if len(parts) == 3 {
		mode = strings.TrimSpace(parts[2])
	}

	if source == "" {
		return "", "", NewUserFacingError("Volume source path is required")
	}
	if filepath.IsAbs(source) {
		return "", "", NewUserFacingError("Volume source must be relative to the resource mount root")
	}

	cleanSource := filepath.Clean(source)
	if cleanSource == "." || cleanSource == ".." || strings.HasPrefix(cleanSource, ".."+string(filepath.Separator)) {
		return "", "", NewUserFacingError("Volume source must stay inside the configured resource mount root")
	}

	if target == "" || !path.IsAbs(target) {
		return "", "", NewUserFacingError("Volume target path must be an absolute container path")
	}
	cleanTarget := path.Clean(target)
	if cleanTarget == "/" {
		return "", "", NewUserFacingError("Volume target path cannot be /")
	}

	if mode != "" && mode != "ro" && mode != "rw" {
		return "", "", NewUserFacingError("Volume mode must be ro or rw")
	}

	hostPath := filepath.Clean(filepath.Join(root, cleanSource))
	if hostPath != root && !strings.HasPrefix(hostPath, root+string(filepath.Separator)) {
		return "", "", NewUserFacingError("Volume source must stay inside the configured resource mount root")
	}

	bind := hostPath + ":" + cleanTarget
	if mode == "ro" {
		bind += ":ro"
	}
	return bind, hostPath, nil
}
