package command

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"tango/internal/domain"
)

var newRandomVolumePrefix = generateRandomVolumePrefix

// ApplyRandomVolumePrefix namespaces relative volume sources and volume_files
// paths with a shared random prefix so component storage does not collide across
// stack instances.
func ApplyRandomVolumePrefix(
	volumes []string,
	volumeFiles []domain.VolumeFileTemplate,
) ([]string, []domain.VolumeFileTemplate, error) {
	if len(volumes) == 0 && len(volumeFiles) == 0 {
		return volumes, volumeFiles, nil
	}

	prefix, err := newRandomVolumePrefix()
	if err != nil {
		return nil, nil, err
	}

	return applyVolumePrefix(prefix, volumes, volumeFiles), applyVolumeFilePrefix(prefix, volumeFiles), nil
}

func generateRandomVolumePrefix() (string, error) {
	buf := make([]byte, 2)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate volume prefix: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func applyVolumePrefix(prefix string, volumes []string, volumeFiles []domain.VolumeFileTemplate) []string {
	if len(volumes) == 0 {
		return nil
	}

	out := make([]string, 0, len(volumes))
	for _, item := range volumes {
		trimmed := strings.TrimSpace(item)
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			out = append(out, item)
			continue
		}
		source := strings.TrimSpace(parts[0])
		if source == "" {
			out = append(out, item)
			continue
		}
		out = append(out, prefix+"-"+source+":"+parts[1])
	}
	return out
}

func applyVolumeFilePrefix(prefix string, volumeFiles []domain.VolumeFileTemplate) []domain.VolumeFileTemplate {
	if len(volumeFiles) == 0 {
		return nil
	}

	out := make([]domain.VolumeFileTemplate, 0, len(volumeFiles))
	for _, item := range volumeFiles {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			out = append(out, item)
			continue
		}
		out = append(out, domain.VolumeFileTemplate{
			Path:    prefix + "-" + path,
			Content: item.Content,
		})
	}
	return out
}
