package domain

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// VolumeFileTemplate describes a file to pre-populate on the host volume path
// before the container starts. Content may contain {{VAR_NAME}} placeholders
// that are replaced with the resource's env var values at start time.
type VolumeFileTemplate struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WriteVolumeFiles writes pre-populated files into host volume paths before
// the container starts. config["volume_files"] must be a list of objects with
// "path" (relative to mountRoot) and "content" (template string) fields.
// {{VAR_NAME}} placeholders in content are replaced with values from env.
func WriteVolumeFiles(cfg map[string]any, mountRoot string, env map[string]string) error {
	if cfg == nil {
		return nil
	}
	raw, ok := cfg["volume_files"]
	if !ok {
		return nil
	}

	root := filepath.Clean(strings.TrimSpace(mountRoot))
	if root == "" || root == "." {
		return NewUserFacingError("Resource mount root is not configured")
	}

	items, err := toVolumeFileList(raw)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := writeOneVolumeFile(item.Path, item.Content, root, env); err != nil {
			return err
		}
	}
	return nil
}

func toVolumeFileList(raw any) ([]VolumeFileTemplate, error) {
	list, ok := raw.([]interface{})
	if !ok {
		return nil, NewUserFacingError("volume_files must be a list of objects")
	}
	result := make([]VolumeFileTemplate, 0, len(list))
	for _, item := range list {
		switch v := item.(type) {
		case map[string]interface{}:
			p, _ := v["path"].(string)
			c, _ := v["content"].(string)
			if strings.TrimSpace(p) == "" {
				return nil, NewUserFacingError("each volume_files entry must have a non-empty path")
			}
			result = append(result, VolumeFileTemplate{Path: p, Content: c})
		case VolumeFileTemplate:
			result = append(result, v)
		default:
			return nil, NewUserFacingError("each volume_files entry must be an object with path and content")
		}
	}
	return result, nil
}

func writeOneVolumeFile(filePath, content, root string, env map[string]string) error {
	clean := filepath.Clean(strings.TrimSpace(filePath))
	if clean == "" || filepath.IsAbs(clean) {
		return NewUserFacingError("volume_files path must be relative to the resource mount root")
	}
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return NewUserFacingError("volume_files path must stay inside the resource mount root")
	}

	hostPath := filepath.Clean(filepath.Join(root, clean))
	if hostPath == root || !strings.HasPrefix(hostPath, root+string(filepath.Separator)) {
		return NewUserFacingError("volume_files path must stay inside the resource mount root")
	}

	content = interpolateEnvVars(content, env)

	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		return fmt.Errorf("prepare directory for volume file %s: %w", filePath, err)
	}
	if err := os.WriteFile(hostPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write volume file %s: %w", filePath, err)
	}
	return nil
}

func interpolateEnvVars(content string, env map[string]string) string {
	if len(env) == 0 {
		return content
	}
	pairs := make([]string, 0, len(env)*2)
	for k, v := range env {
		pairs = append(pairs, "{{"+k+"}}", v)
	}
	return strings.NewReplacer(pairs...).Replace(content)
}

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

	items, err := toStringList(raw)
	if err != nil {
		return ResourceMounts{}, err
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
		bind, hostPath, err := resolveResourceMount(strings.TrimSpace(item), root)
		if err != nil {
			return ResourceMounts{}, err
		}
		result.Binds = append(result.Binds, bind)
		result.HostPaths = append(result.HostPaths, hostPath)
	}

	return result, nil
}

func toStringList(raw any) ([]string, error) {
	switch items := raw.(type) {
	case []string:
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, item)
		}
		return out, nil
	case []interface{}:
		out := make([]string, 0, len(items))
		for _, item := range items {
			value, ok := item.(string)
			if !ok {
				return nil, NewUserFacingError("Resource volumes must be strings in source:target[:mode] format")
			}
			out = append(out, value)
		}
		return out, nil
	default:
		return nil, NewUserFacingError("Resource volumes must be a list of mount definitions")
	}
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
