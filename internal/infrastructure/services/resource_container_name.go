package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"tango/internal/domain"
)

var nonContainerNameChars = regexp.MustCompile(`[^a-z0-9._-]+`)

func buildUniqueContainerName(ctx context.Context, dockerRepo domain.DockerRepository, resource *domain.Resource) (string, error) {
	base := normalizeContainerName(resource.Name)
	if base == "" {
		base = fmt.Sprintf("resource-%s", shortResourceID(resource.ID))
	}

	if dockerRepo == nil {
		return base, nil
	}

	containers, err := dockerRepo.ListContainers(ctx, true)
	if err != nil {
		return "", fmt.Errorf("list containers for name conflict check: %w", err)
	}

	used := make(map[string]struct{}, len(containers))
	for _, ct := range containers {
		name := strings.TrimSpace(ct.Name)
		if name != "" {
			used[name] = struct{}{}
		}
	}

	if _, exists := used[base]; !exists {
		return base, nil
	}

	for i := 2; i <= 99; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, exists := used[candidate]; !exists {
			return candidate, nil
		}
	}

	fallback := fmt.Sprintf("%s-%s", base, shortResourceID(resource.ID))
	if _, exists := used[fallback]; !exists {
		return fallback, nil
	}

	for i := 2; i <= 99; i++ {
		candidate := fmt.Sprintf("%s-%s-%d", base, shortResourceID(resource.ID), i)
		if _, exists := used[candidate]; !exists {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not allocate unique container name for resource %s", resource.ID)
}

func normalizeContainerName(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = nonContainerNameChars.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-.")
	normalized = collapseDashes(normalized)
	return normalized
}

func collapseDashes(value string) string {
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return value
}

func shortResourceID(value string) string {
	clean := strings.TrimSpace(value)
	if len(clean) <= 8 {
		return clean
	}
	return clean[:8]
}
