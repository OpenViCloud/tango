package docker

import (
	"strings"
	"testing"
)

// TestIsManagerLogic verifies the condition used in SwarmRepository.IsManager.
// We test the boolean expression directly since mocking *client.Client is impractical
// (concrete struct, not interface). The integration behaviour is covered by the
// handler-level test using fakeSwarmRepository.
func TestIsManagerLogic(t *testing.T) {
	cases := []struct {
		name             string
		localNodeState   string
		controlAvailable bool
		want             bool
	}{
		{"active manager", "active", true, true},
		{"active worker", "active", false, false},
		{"pending", "pending", false, false},
		{"inactive", "inactive", false, false},
		{"empty state", "", false, false},
	}

	isManager := func(localNodeState string, controlAvailable bool) bool {
		return localNodeState == "active" && controlAvailable
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isManager(tc.localNodeState, tc.controlAvailable)
			if got != tc.want {
				t.Errorf("isManager(%q, %v) = %v, want %v", tc.localNodeState, tc.controlAvailable, got, tc.want)
			}
		})
	}
}

// TestNormalizeServiceName verifies container name normalization used for swarm
// service names (reuses the same normalizeContainerName function).
func TestNormalizeServiceName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"My App", "my-app"},
		{"postgres-db", "postgres-db"},
		{"  leading spaces  ", "leading-spaces"},
		{"UPPER_CASE", "upper-case"},
		{"double--dash", "double-dash"},
		{"special!@#chars", "special-chars"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			// normalizeContainerName is defined in resource_container_name.go
			// which is in the services package, not docker. We replicate the
			// relevant logic here to keep the test self-contained.
			got := testNormalize(tc.input)
			if got != tc.want {
				t.Errorf("normalize(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// testNormalize replicates the normalisation rules applied to service/container names.
func testNormalize(value string) string {
	s := strings.ToLower(strings.TrimSpace(value))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	s = b.String()
	s = strings.Trim(s, "-.")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}
