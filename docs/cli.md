# CLI

The Tango CLI is a native binary installed on the host machine. It manages the Docker Compose stack, monitors service health, and auto-restarts unhealthy containers.

The API and all workloads run inside Docker. The CLI runs outside Docker as the control plane.

```
┌──────────────────────────────────────────────┐
│  Host machine                                │
│                                              │
│  tango CLI (native binary)                   │
│    ├── daemon (health check loop)            │
│    ├── service (manage containers)           │
│    └── status / uninstall                    │
│                                              │
│  Docker                                      │
│    ├── app         (API + frontend)          │
│    ├── db          (PostgreSQL)              │
│    ├── traefik     (reverse proxy)           │
│    ├── buildkitd   (image builds)            │
│    └── backup-runner (dump/restore)          │
└──────────────────────────────────────────────┘
```

## Installation

```bash
# Option 1: Go install
go install tango/cmd/cli@latest

# Option 2: Download binary from GitHub Releases
curl -fsSL https://github.com/time-groups/tango-cloud/releases/download/cli-latest/tango-linux-amd64 -o tango
chmod +x tango
sudo install -m 0755 tango /usr/local/bin/tango

# Option 3: Homebrew (planned)
# brew install tango-cloud/tap/tango
```

For ARM Linux hosts, replace `tango-linux-amd64` with `tango-linux-arm64`.

## File Layout

The CLI stores all its files under `~/.config/tango/`:

```
~/.config/tango/
├── daemon.json          ← daemon configuration
├── daemon.pid           ← PID of running daemon process
├── daemon-status.json   ← latest health check results
└── daemon.log           ← daemon process logs
```

### File reference

| File | Written by | Read by | Purpose |
|------|-----------|---------|---------|
| `daemon.json` | User (manual edit) | `tango daemon run` | Check interval, max retries, compose file path, monitored services |
| `daemon.pid` | `tango daemon run` | `tango daemon start/stop/status` | Track daemon process lifecycle |
| `daemon-status.json` | `tango daemon run` (every check cycle) | `tango daemon status` | Service health table: state, health, restart count, exhausted flag |
| `daemon.log` | `tango daemon run` | Admin (manual `cat`/`tail`) | Debug log for daemon actions: restarts, errors, Docker connectivity |

### Log sources

There are three separate log streams. They come from different sources and serve different purposes:

| What | Source | How to view | Rotation |
|------|--------|-------------|----------|
| **App/service logs** | Docker container stdout/stderr | `tango service logs <name>` | Docker `json-file` driver: `max-size` + `max-file` in docker-compose.yml |
| **Daemon log** | Daemon health check process | `cat ~/.config/tango/daemon.log` | Manual or logrotate (daemon writes JSON lines) |
| **API server log** | Go app inside container | Web dashboard or `docker compose logs app` | Configured via `LOG_*` env vars (see [Configuration](configuration.md)) |

`tango service logs` and `tango daemon status` read from completely independent sources:

```
tango service logs app     → Docker Engine API → container stdout/stderr
tango daemon status        → ~/.config/tango/daemon-status.json (local file)
```

## Commands

### Daemon

The daemon runs a background health check loop. Every 30 seconds (configurable), it:

1. Checks if Docker daemon is reachable (`docker info`)
2. Lists all compose services (`docker compose ps`)
3. Restarts any exited, dead, or unhealthy containers
4. Tracks retry count per service (max 3 within 5 minutes by default)
5. Optionally checks HTTP health endpoint
6. Writes results to `daemon-status.json`

```bash
# Start daemon in background
tango daemon start

# Check health status
tango daemon status

# Stop daemon
tango daemon stop

# Install as system service (auto-start on boot)
tango daemon install      # macOS: launchd, Linux: systemd
tango daemon uninstall
```

`tango daemon status` output:

```
Daemon running (PID: 12345)
Last check: 2026-04-03 14:30:05
State: ok

SERVICE              STATE        HEALTH       RETRIES  EXHAUSTED
-------              -----        ------       -------  ---------
app                  running      healthy      0
db                   running      healthy      0
traefik              running      none         0
buildkitd            running      none         0
backup-runner        running      none         0
```

### Health check flow

```
┌─────────────────┐
│  Daemon loop     │
│  (every 30s)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────────┐
│ Docker alive?   │──NO─▶│ Log error, skip cycle │
│ (docker info)   │     └──────────────────────┘
└────────┬────────┘
         │ YES
         ▼
┌─────────────────┐
│ List services   │
│ (compose ps)    │
└────────┬────────┘
         │
         ▼
┌─────────────────────────┐
│ For each service:       │
│                         │
│ running + healthy → OK  │
│ running + unhealthy     │──▶ restart (up to max_retries)
│ exited / dead           │──▶ restart (up to max_retries)
│                         │
│ Retry exhausted?        │──▶ log critical, stop retrying
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│ HTTP health check       │
│ (GET /api/status)       │
│ Optional, for app-level │
│ liveness detection      │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│ Write daemon-status.json│
└─────────────────────────┘
```

### Retry logic

- Each service has an independent retry counter
- Default: max 3 restarts within a 5-minute window
- If the service recovers and stays healthy past the cooldown window, the counter resets
- If retries are exhausted, the daemon stops trying and logs a critical warning
- This prevents restart loops (e.g., a service that crashes immediately on start)

### Service

Direct service management commands. These work without the daemon running.

```bash
# List all services
tango service list

# Detailed status for one service
tango service status app

# Restart / stop / start a service
tango service restart app
tango service stop app
tango service start app

# View container logs
tango service logs app
tango service logs app --tail 100

# Specify compose file (defaults to daemon.json config or ./docker-compose.yml)
tango service list -f /path/to/docker-compose.yml
tango service list -p my-project
```

`tango service list` output:

```
SERVICE              STATE        HEALTH       IMAGE           PORTS
-------              -----        ------       -----           -----
app                  running      healthy      timegroups/t... 0.0.0.0:8080->8080/tcp
db                   running      healthy      postgres:16-...
traefik              running      none         traefik:v3.6    0.0.0.0:80->80/tcp (+1)
buildkitd            running      none         moby/buildki...
backup-runner        running      none         timegroups/t... 0.0.0.0:8081->8081/tcp
```

### Status

```bash
tango status         # check API / stack status endpoint
```

### Other

```bash
tango version        # show CLI version
tango uninstall      # remove CLI, daemon service, and local config
tango uninstall --purge
```

## Configuration

### Daemon config (`~/.config/tango/daemon.json`)

```json
{
  "driver": "compose",
  "check_interval": "30s",
  "max_retries": 3,
  "retry_backoff": "10s",
  "retry_cooldown": "5m",
  "compose_file": "/opt/tango/docker-compose.yml",
  "project_name": "tango",
  "health_url": "http://localhost:8080/api/status",
  "services": ["app", "db", "traefik", "buildkitd", "backup-runner"]
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `driver` | `compose` | Orchestrator backend. Future: `k3s`, `swarm`, `nomad` |
| `check_interval` | `30s` | How often the daemon checks service health |
| `max_retries` | `3` | Max restart attempts per service within the cooldown window |
| `retry_backoff` | `10s` | Wait time between restart attempts |
| `retry_cooldown` | `5m` | Window for retry counter. Resets if service stays healthy beyond this |
| `compose_file` | `""` | Path to docker-compose.yml. Empty = Docker Compose default |
| `project_name` | `""` | Compose project name. Empty = directory name |
| `health_url` | `http://localhost:8080/api/status` | HTTP endpoint for app-level health check |
| `services` | `[]` | Services to monitor. Empty = all services in compose file |

### Environment variable overrides

| Variable | Overrides |
|----------|-----------|
| `TANGO_DRIVER` | `driver` |
| `TANGO_CHECK_INTERVAL` | `check_interval` |
| `TANGO_COMPOSE_FILE` | `compose_file` |
| `TANGO_PROJECT_NAME` | `project_name` |
| `TANGO_HEALTH_URL` | `health_url` |

### System service

`tango daemon install` creates a system service so the daemon starts automatically on boot and restarts on failure:

- **macOS**: `~/Library/LaunchAgents/com.tango.daemon.plist` (launchd)
- **Linux**: `~/.config/systemd/user/tango-daemon.service` (systemd user unit)

## Driver Architecture

The CLI uses a pluggable driver interface for orchestrator backends. Phase 1 ships with the Docker Compose driver. Future drivers can be added without changing the CLI commands.

```
CLI commands (daemon, service)
       │
       ▼
Driver interface
       │
       ├── compose (Phase 1) ← shells out to `docker compose`
       ├── k3s     (planned) ← kubectl / k3s API
       ├── swarm   (planned) ← docker swarm API
       └── nomad   (planned) ← nomad API
```

All drivers implement the same interface:

```go
type Driver interface {
    Info(ctx)                          (DriverInfo, error)
    Ping(ctx)                         error
    ListServices(ctx)                 ([]ServiceStatus, error)
    ServiceStatus(ctx, name)          (ServiceStatus, error)
    RestartService(ctx, name)         error
    StartService(ctx, name)           error
    StopService(ctx, name)            error
    ServiceLogs(ctx, name, tail)      (io.ReadCloser, error)
    Close()                           error
}
```

## Troubleshooting

**Daemon won't start**
```bash
# Check if already running
tango daemon status

# Check PID file
cat ~/.config/tango/daemon.pid

# Check logs
tail -20 ~/.config/tango/daemon.log
```

**Service keeps restarting (exhausted retries)**
```bash
# Check why the service is crashing
tango service logs <name> --tail 100

# Reset retries by restarting daemon
tango daemon stop
tango daemon start
```

**Docker daemon not reachable**
```bash
# Verify Docker is running
docker info

# Check daemon log for connectivity errors
tail -20 ~/.config/tango/daemon.log
```
