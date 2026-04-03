package compose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"tango/internal/orchestrator"
)

// Driver implements orchestrator.Driver using docker compose CLI.
type Driver struct {
	composeFile string
	projectName string
}

// New creates a new Compose driver.
func New(composeFile, projectName string) *Driver {
	return &Driver{
		composeFile: composeFile,
		projectName: projectName,
	}
}

func (d *Driver) baseArgs() []string {
	var args []string
	if d.composeFile != "" {
		args = append(args, "-f", d.composeFile)
	}
	if d.projectName != "" {
		args = append(args, "-p", d.projectName)
	}
	return args
}

func (d *Driver) run(ctx context.Context, args ...string) (string, string, error) {
	fullArgs := append(d.baseArgs(), args...)
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, fullArgs...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (d *Driver) Info(ctx context.Context) (orchestrator.DriverInfo, error) {
	info := orchestrator.DriverInfo{Name: "compose"}

	stdout, _, err := d.run(ctx, "version", "--short")
	if err != nil {
		info.Error = fmt.Sprintf("docker compose not available: %v", err)
		return info, nil
	}

	info.Version = strings.TrimSpace(stdout)
	info.Ready = true
	return info, nil
}

func (d *Driver) Ping(ctx context.Context) error {
	// Check Docker daemon
	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker daemon not running: %w", err)
	}

	// Check compose file is valid
	_, stderr, err := d.run(ctx, "config", "--quiet")
	if err != nil {
		return fmt.Errorf("invalid compose file: %s", strings.TrimSpace(stderr))
	}
	return nil
}

func (d *Driver) Up(ctx context.Context) error {
	_, stderr, err := d.run(ctx, "up", "-d")
	if err != nil {
		return fmt.Errorf("docker compose up failed: %s: %w", strings.TrimSpace(stderr), err)
	}
	return nil
}

// composePS is the JSON output from `docker compose ps --format json`.
type composePS struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
	Service    string `json:"Service"`
	State      string `json:"State"`
	Health     string `json:"Health"`
	ExitCode   int    `json:"ExitCode"`
	Image      string `json:"Image"`
	Publishers []struct {
		URL           string `json:"URL"`
		TargetPort    int    `json:"TargetPort"`
		PublishedPort int    `json:"PublishedPort"`
		Protocol      string `json:"Protocol"`
	} `json:"Publishers"`
	CreatedAt string `json:"CreatedAt"`
}

func parseComposePS(output string) ([]orchestrator.ServiceStatus, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var services []orchestrator.ServiceStatus

	// docker compose ps --format json outputs NDJSON (one JSON object per line)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var ps composePS
		if err := json.Unmarshal([]byte(line), &ps); err != nil {
			// Try parsing as JSON array (some versions output an array)
			var psArray []composePS
			if err2 := json.Unmarshal([]byte(line), &psArray); err2 != nil {
				return nil, fmt.Errorf("failed to parse compose ps output: %w", err)
			}
			for _, p := range psArray {
				services = append(services, toServiceStatus(p))
			}
			continue
		}
		services = append(services, toServiceStatus(ps))
	}

	return services, nil
}

func toServiceStatus(ps composePS) orchestrator.ServiceStatus {
	var ports []string
	for _, p := range ps.Publishers {
		if p.PublishedPort > 0 {
			ports = append(ports, fmt.Sprintf("%s:%d->%d/%s", p.URL, p.PublishedPort, p.TargetPort, p.Protocol))
		}
	}

	health := ps.Health
	if health == "" {
		health = "none"
	}

	var uptime time.Duration
	if ps.CreatedAt != "" {
		if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", ps.CreatedAt); err == nil {
			uptime = time.Since(t)
		}
	}

	return orchestrator.ServiceStatus{
		Name:        ps.Service,
		State:       ps.State,
		Health:      health,
		ContainerID: ps.ID,
		Image:       ps.Image,
		Uptime:      uptime,
		ExitCode:    ps.ExitCode,
		Ports:       ports,
	}
}

func (d *Driver) ListServices(ctx context.Context) ([]orchestrator.ServiceStatus, error) {
	stdout, stderr, err := d.run(ctx, "ps", "-a", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("docker compose ps failed: %s: %w", strings.TrimSpace(stderr), err)
	}
	return parseComposePS(stdout)
}

func (d *Driver) WaitReady(ctx context.Context, services []string) error {
	required := services
	if len(required) == 0 {
		required = append([]string(nil), orchestrator.DefaultServices...)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		current, err := d.ListServices(ctx)
		if err == nil && servicesReady(current, required) {
			return nil
		}

		select {
		case <-ctx.Done():
			if err != nil {
				return fmt.Errorf("wait ready: %w", err)
			}
			return fmt.Errorf("wait ready: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func servicesReady(current []orchestrator.ServiceStatus, required []string) bool {
	if len(current) == 0 || len(required) == 0 {
		return false
	}

	seen := make(map[string]orchestrator.ServiceStatus, len(current))
	for _, svc := range current {
		seen[svc.Name] = svc
	}

	for _, name := range required {
		svc, ok := seen[name]
		if !ok {
			return false
		}
		if svc.State != "running" {
			return false
		}
		if svc.Health != "healthy" && svc.Health != "none" {
			return false
		}
	}

	return true
}

func (d *Driver) ServiceStatus(ctx context.Context, name string) (orchestrator.ServiceStatus, error) {
	stdout, stderr, err := d.run(ctx, "ps", "-a", "--format", "json", name)
	if err != nil {
		return orchestrator.ServiceStatus{}, fmt.Errorf("docker compose ps %s failed: %s: %w", name, strings.TrimSpace(stderr), err)
	}

	services, err := parseComposePS(stdout)
	if err != nil {
		return orchestrator.ServiceStatus{}, err
	}
	if len(services) == 0 {
		return orchestrator.ServiceStatus{}, fmt.Errorf("service %q not found", name)
	}
	return services[0], nil
}

func (d *Driver) RestartService(ctx context.Context, name string) error {
	_, stderr, err := d.run(ctx, "restart", name)
	if err != nil {
		return fmt.Errorf("restart %s failed: %s: %w", name, strings.TrimSpace(stderr), err)
	}
	return nil
}

func (d *Driver) StartService(ctx context.Context, name string) error {
	_, stderr, err := d.run(ctx, "up", "-d", name)
	if err != nil {
		return fmt.Errorf("start %s failed: %s: %w", name, strings.TrimSpace(stderr), err)
	}
	return nil
}

func (d *Driver) StopService(ctx context.Context, name string) error {
	_, stderr, err := d.run(ctx, "stop", name)
	if err != nil {
		return fmt.Errorf("stop %s failed: %s: %w", name, strings.TrimSpace(stderr), err)
	}
	return nil
}

func (d *Driver) ServiceLogs(ctx context.Context, name string, tail int) (io.ReadCloser, error) {
	args := append(d.baseArgs(), "logs", "--no-color", "--tail", fmt.Sprintf("%d", tail), name)
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("logs %s failed: %w", name, err)
	}

	// Return a reader that also waits for the command to finish
	return &cmdReadCloser{ReadCloser: stdout, cmd: cmd}, nil
}

func (d *Driver) Down(ctx context.Context, removeVolumes bool) error {
	args := []string{"down", "--remove-orphans"}
	if removeVolumes {
		args = append(args, "-v", "--rmi", "all")
	}
	_, stderr, err := d.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("docker compose down failed: %s: %w", strings.TrimSpace(stderr), err)
	}
	return nil
}

func (d *Driver) Close() error {
	return nil
}

type cmdReadCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (c *cmdReadCloser) Close() error {
	c.ReadCloser.Close()
	return c.cmd.Wait()
}
