package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"tango/internal/orchestrator"
)

// Daemon runs the health-check loop.
type Daemon struct {
	driver  orchestrator.Driver
	config  orchestrator.DaemonConfig
	logger  *slog.Logger
	retries map[string]*retryState
	mu      sync.Mutex
	started time.Time
}

type retryState struct {
	count     int
	firstAt   time.Time
	lastAt    time.Time
	exhausted bool
}

// New creates a new Daemon instance.
func New(driver orchestrator.Driver, config orchestrator.DaemonConfig, logger *slog.Logger) *Daemon {
	return &Daemon{
		driver:  driver,
		config:  config,
		logger:  logger,
		retries: make(map[string]*retryState),
		started: time.Now(),
	}
}

// Run starts the health-check loop. Blocks until context is cancelled or signal received.
func (d *Daemon) Run(ctx context.Context) error {
	// Write PID file
	if err := writePID(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	defer removePID()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	registerShutdownSignals(sigCh)
	defer signal.Stop(sigCh)
	go func() {
		sig := <-sigCh
		d.logger.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	d.logger.Info("daemon started",
		"check_interval", d.config.CheckInterval,
		"max_retries", d.config.MaxRetries,
		"compose_file", d.config.ComposeFile,
	)

	if err := d.bootstrap(ctx); err != nil {
		d.logger.Error("bootstrap failed", "err", err)
	}

	d.check(ctx)

	ticker := time.NewTicker(d.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("daemon stopped")
			return nil
		case <-ticker.C:
			d.check(ctx)
		}
	}
}

func (d *Daemon) bootstrap(ctx context.Context) error {
	d.writeStatus(nil, "bootstrapping")

	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	for {
		if err := d.driver.Ping(waitCtx); err == nil {
			break
		} else {
			d.logger.Warn("waiting for docker", "err", err)
		}

		select {
		case <-waitCtx.Done():
			d.writeStatus(nil, "docker_down")
			return fmt.Errorf("docker not ready: %w", waitCtx.Err())
		case <-time.After(3 * time.Second):
		}
	}

	upCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := d.driver.Up(upCtx); err != nil {
		d.writeStatus(nil, "degraded")
		return fmt.Errorf("start stack: %w", err)
	}

	readyCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if err := d.driver.WaitReady(readyCtx, d.config.Services); err != nil {
		d.writeStatus(nil, "degraded")
		return fmt.Errorf("wait ready: %w", err)
	}

	return nil
}

func (d *Daemon) check(ctx context.Context) {
	d.logger.Debug("running health check")

	// 1. Check Docker daemon
	if err := d.driver.Ping(ctx); err != nil {
		d.logger.Error("docker daemon unreachable", "err", err)
		d.writeStatus(nil, "docker_down")
		return
	}

	// 2. Check all services
	services, err := d.driver.ListServices(ctx)
	if err != nil {
		d.logger.Error("failed to list services", "err", err)
		d.writeStatus(nil, "list_failed")
		return
	}

	// Filter services if configured
	if len(d.config.Services) > 0 {
		filtered := make([]orchestrator.ServiceStatus, 0)
		serviceSet := make(map[string]bool)
		for _, s := range d.config.Services {
			serviceSet[s] = true
		}
		for _, s := range services {
			if serviceSet[s.Name] {
				filtered = append(filtered, s)
			}
		}
		services = filtered
	}

	// 3. Check each service and restart if needed
	reports := make([]ServiceReport, 0, len(services))
	for _, svc := range services {
		report := d.checkService(ctx, svc)
		reports = append(reports, report)
	}

	// 4. Optional HTTP health check
	if d.config.HealthURL != "" {
		d.checkHTTPHealth(ctx)
	}

	// 5. Write status file
	d.writeStatus(reports, "ok")
}

func (d *Daemon) checkService(ctx context.Context, svc orchestrator.ServiceStatus) ServiceReport {
	// Evaluate under a single lock
	report, needsRestart, method := d.evaluateService(svc)

	if needsRestart {
		d.logger.Info("restarting service",
			"service", svc.Name,
			"attempt", report.RestartCount,
			"max", d.config.MaxRetries,
			"method", method,
		)

		var restartErr error
		if method == "start" {
			restartErr = d.driver.StartService(ctx, svc.Name)
		} else {
			restartErr = d.driver.RestartService(ctx, svc.Name)
		}

		if restartErr != nil {
			d.logger.Error("failed to restart service", "service", svc.Name, "err", restartErr)
		} else {
			d.logger.Info("service restarted", "service", svc.Name)
			report.LastRestart = time.Now().Format(time.RFC3339)
		}
	}

	return report
}

// evaluateService determines whether a service needs restart. Must be called
// from a single goroutine or protected externally — uses d.mu internally.
func (d *Daemon) evaluateService(svc orchestrator.ServiceStatus) (report ServiceReport, needsRestart bool, method string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rs, exists := d.retries[svc.Name]
	if !exists {
		rs = &retryState{}
		d.retries[svc.Name] = rs
	}

	report = ServiceReport{
		Name:         svc.Name,
		State:        svc.State,
		Health:       svc.Health,
		RestartCount: rs.count,
		Exhausted:    rs.exhausted,
	}

	switch svc.State {
	case "running":
		if svc.Health == "unhealthy" {
			d.logger.Warn("service unhealthy", "service", svc.Name)
			needsRestart = true
			method = "restart"
		} else {
			// Service is healthy — reset retry counter if cooldown passed
			if rs.count > 0 && time.Since(rs.firstAt) > d.config.RetryCooldown {
				rs.count = 0
				rs.exhausted = false
				d.logger.Info("retry counter reset", "service", svc.Name)
			}
			return
		}
	case "exited", "dead":
		d.logger.Warn("service not running", "service", svc.Name, "state", svc.State, "exit_code", svc.ExitCode)
		needsRestart = true
		method = "start"
	default:
		return
	}

	// Check if retries are exhausted
	if rs.exhausted {
		needsRestart = false
		d.logger.Error("service restart exhausted, skipping", "service", svc.Name, "retries", rs.count)
		return
	}

	// Reset counter if cooldown window has passed since first retry
	if rs.count > 0 && time.Since(rs.firstAt) > d.config.RetryCooldown {
		rs.count = 0
	}

	if rs.count >= d.config.MaxRetries {
		rs.exhausted = true
		report.Exhausted = true
		needsRestart = false
		d.logger.Error("service exceeded max retries",
			"service", svc.Name,
			"max_retries", d.config.MaxRetries,
			"window", d.config.RetryCooldown,
		)
		return
	}

	rs.count++
	if rs.count == 1 {
		rs.firstAt = time.Now()
	}
	rs.lastAt = time.Now()
	report.RestartCount = rs.count
	return
}

func (d *Daemon) checkHTTPHealth(ctx context.Context) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.config.HealthURL, nil)
	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		d.logger.Warn("HTTP health check failed", "url", d.config.HealthURL, "err", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 500 {
		d.logger.Warn("HTTP health check returned error", "url", d.config.HealthURL, "status", resp.StatusCode)
	}
}

func writePID() error {
	pidPath := orchestrator.PIDPath()
	if err := os.MkdirAll(orchestrator.ConfigDir(), 0700); err != nil {
		return err
	}
	exe, _ := os.Executable()
	exe, _ = filepath.EvalSymlinks(exe)
	content := fmt.Sprintf("%d:%s", os.Getpid(), exe)
	return os.WriteFile(pidPath, []byte(content), 0600)
}

func removePID() {
	os.Remove(orchestrator.PIDPath())
}

// ReadPID reads the daemon PID from the PID file. Returns 0 if not found.
func ReadPID() int {
	data, err := os.ReadFile(orchestrator.PIDPath())
	if err != nil {
		return 0
	}
	parts := strings.SplitN(string(data), ":", 2)
	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return pid
}

// IsRunning checks if the daemon process is alive and is actually a tango process.
func IsRunning() bool {
	data, err := os.ReadFile(orchestrator.PIDPath())
	if err != nil {
		return false
	}
	parts := strings.SplitN(string(data), ":", 2)
	pid, err := strconv.Atoi(parts[0])
	if err != nil || pid == 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if !processExists(process) {
		return false
	}

	// Verify the PID belongs to a tango process (guards against PID reuse)
	if len(parts) > 1 {
		exe, _ := os.Executable()
		exe, _ = filepath.EvalSymlinks(exe)
		if parts[1] != exe {
			return false
		}
	}
	return true
}
