package orchestrator

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

var DefaultServices = []string{"app", "traefik", "db", "buildkitd", "backup-runner"}

// DaemonConfig holds configuration for the health-check daemon.
type DaemonConfig struct {
	Driver        string        `json:"driver"`         // "compose" (default)
	CheckInterval time.Duration `json:"check_interval"` // default 30s
	MaxRetries    int           `json:"max_retries"`    // per service before giving up, default 3
	RetryBackoff  time.Duration `json:"retry_backoff"`  // wait between retries, default 10s
	RetryCooldown time.Duration `json:"retry_cooldown"` // reset retry counter after this window, default 5m
	ComposeFile   string        `json:"compose_file"`   // path to docker-compose.yml
	ProjectName   string        `json:"project_name"`   // compose project name
	HealthURL     string        `json:"health_url"`     // HTTP health endpoint, e.g. http://localhost:8080/api/status
	Services      []string      `json:"services"`       // services to monitor (empty = all)
}

// configJSON is the on-disk representation with string durations.
type configJSON struct {
	Driver        string   `json:"driver"`
	CheckInterval string   `json:"check_interval"`
	MaxRetries    int      `json:"max_retries"`
	RetryBackoff  string   `json:"retry_backoff"`
	RetryCooldown string   `json:"retry_cooldown"`
	ComposeFile   string   `json:"compose_file"`
	ProjectName   string   `json:"project_name"`
	HealthURL     string   `json:"health_url"`
	Services      []string `json:"services"`
}

// DefaultConfig returns a DaemonConfig with sensible defaults.
func DefaultConfig() DaemonConfig {
	return DaemonConfig{
		Driver:        "compose",
		CheckInterval: 30 * time.Second,
		MaxRetries:    3,
		RetryBackoff:  10 * time.Second,
		RetryCooldown: 5 * time.Minute,
		HealthURL:     "http://localhost:8080/api/status",
		Services:      append([]string(nil), DefaultServices...),
	}
}

// ConfigDir returns the tango config directory.
func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tango")
}

// ConfigPath returns the path to the daemon config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "daemon.json")
}

// PIDPath returns the path to the daemon PID file.
func PIDPath() string {
	return filepath.Join(ConfigDir(), "daemon.pid")
}

// StatusPath returns the path to the daemon status file.
func StatusPath() string {
	return filepath.Join(ConfigDir(), "daemon-status.json")
}

// LogPath returns the path to the daemon log file.
func LogPath() string {
	return filepath.Join(ConfigDir(), "daemon.log")
}

// LoadConfig loads the daemon config from disk, applying defaults and env overrides.
func LoadConfig() DaemonConfig {
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigPath())
	if err == nil {
		var raw configJSON
		if err := json.Unmarshal(data, &raw); err != nil {
			slog.Warn("invalid daemon config file, using defaults", "path", ConfigPath(), "err", fmt.Sprintf("%v", err))
		} else {
			if raw.Driver != "" {
				cfg.Driver = raw.Driver
			}
			if d, err := time.ParseDuration(raw.CheckInterval); err == nil {
				cfg.CheckInterval = d
			}
			if raw.MaxRetries > 0 {
				cfg.MaxRetries = raw.MaxRetries
			}
			if d, err := time.ParseDuration(raw.RetryBackoff); err == nil {
				cfg.RetryBackoff = d
			}
			if d, err := time.ParseDuration(raw.RetryCooldown); err == nil {
				cfg.RetryCooldown = d
			}
			if raw.ComposeFile != "" {
				cfg.ComposeFile = raw.ComposeFile
			}
			if raw.ProjectName != "" {
				cfg.ProjectName = raw.ProjectName
			}
			if raw.HealthURL != "" {
				cfg.HealthURL = raw.HealthURL
			}
			if len(raw.Services) > 0 {
				cfg.Services = raw.Services
			}
		}
	}

	// Environment variable overrides
	if v := os.Getenv("TANGO_DRIVER"); v != "" {
		cfg.Driver = v
	}
	if v := os.Getenv("TANGO_CHECK_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.CheckInterval = d
		}
	}
	if v := os.Getenv("TANGO_COMPOSE_FILE"); v != "" {
		cfg.ComposeFile = v
	}
	if v := os.Getenv("TANGO_PROJECT_NAME"); v != "" {
		cfg.ProjectName = v
	}
	if v := os.Getenv("TANGO_HEALTH_URL"); v != "" {
		cfg.HealthURL = v
	}

	return cfg
}

// SaveConfig writes the daemon config to disk.
func SaveConfig(cfg DaemonConfig) error {
	raw := configJSON{
		Driver:        cfg.Driver,
		CheckInterval: cfg.CheckInterval.String(),
		MaxRetries:    cfg.MaxRetries,
		RetryBackoff:  cfg.RetryBackoff.String(),
		RetryCooldown: cfg.RetryCooldown.String(),
		ComposeFile:   cfg.ComposeFile,
		ProjectName:   cfg.ProjectName,
		HealthURL:     cfg.HealthURL,
		Services:      cfg.Services,
	}

	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0600)
}
