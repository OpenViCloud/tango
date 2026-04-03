package daemon

import (
	"encoding/json"
	"os"
	"time"

	"tango/internal/orchestrator"
)

// Status represents the daemon's current state, written to disk each check cycle.
type Status struct {
	Running   bool            `json:"running"`
	PID       int             `json:"pid"`
	StartedAt time.Time       `json:"started_at"`
	LastCheck time.Time       `json:"last_check"`
	State     string          `json:"state"` // "bootstrapping", "ok", "degraded", "docker_down", "list_failed"
	Services  []ServiceReport `json:"services"`
}

// ServiceReport holds the health status for a single service.
type ServiceReport struct {
	Name         string `json:"name"`
	State        string `json:"state"`
	Health       string `json:"health"`
	RestartCount int    `json:"restart_count"`
	LastRestart  string `json:"last_restart,omitempty"`
	Exhausted    bool   `json:"exhausted"`
}

func (d *Daemon) writeStatus(reports []ServiceReport, state string) {
	status := Status{
		Running:   true,
		PID:       os.Getpid(),
		StartedAt: d.started,
		LastCheck: time.Now(),
		State:     state,
		Services:  reports,
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		d.logger.Error("failed to marshal status", "err", err)
		return
	}

	if err := os.MkdirAll(orchestrator.ConfigDir(), 0700); err != nil {
		d.logger.Error("failed to create config dir", "err", err)
		return
	}

	if err := os.WriteFile(orchestrator.StatusPath(), data, 0600); err != nil {
		d.logger.Error("failed to write status file", "err", err)
	}
}

// ReadStatus reads the daemon status from disk.
func ReadStatus() (*Status, error) {
	data, err := os.ReadFile(orchestrator.StatusPath())
	if err != nil {
		return nil, err
	}

	var status Status
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}
