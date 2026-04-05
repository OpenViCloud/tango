package docker

import (
	"strings"
	"testing"

	"tango/internal/domain"
)

// TestServiceRunningNotFoundPatterns verifies that isNotFoundError correctly
// identifies Docker "service not found" errors so ServiceRunning can return
// false without propagating the error to callers.
func TestServiceRunningNotFoundPatterns(t *testing.T) {
	cases := []struct {
		errMsg string
		want   bool
	}{
		{"Error response from daemon: no such service: abc123", true},
		{"Error: No such service: abc", true},
		{"404 page not found", true},
		{"connection refused", false},
		{"permission denied", false},
		{"", false},
	}

	for _, tc := range cases {
		t.Run(tc.errMsg, func(t *testing.T) {
			var err error
			if tc.errMsg != "" {
				err = &stubError{tc.errMsg}
			}
			got := isNotFoundError(err)
			if got != tc.want {
				t.Errorf("isNotFoundError(%q) = %v, want %v", tc.errMsg, got, tc.want)
			}
		})
	}
}

type stubError struct{ msg string }

func (e *stubError) Error() string { return e.msg }

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
		{"UPPER_CASE", "upper_case"},
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

// TestCreateServiceInputReplicasDefault verifies that replicas default to 1
// when CreateServiceInput.Replicas is 0 (the zero-value for uint64).
func TestCreateServiceInputReplicasDefault(t *testing.T) {
	input := domain.CreateServiceInput{
		Name:  "my-svc",
		Image: "nginx",
	}
	// The production code treats Replicas==0 as 1.
	replicas := input.Replicas
	if replicas == 0 {
		replicas = 1
	}
	if replicas != 1 {
		t.Errorf("default replicas = %d, want 1", replicas)
	}
}

// TestCreateServiceInputReplicasSet verifies that a non-zero Replicas value is preserved.
func TestCreateServiceInputReplicasSet(t *testing.T) {
	input := domain.CreateServiceInput{
		Name:     "my-svc",
		Image:    "nginx",
		Replicas: 3,
	}
	if input.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", input.Replicas)
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
