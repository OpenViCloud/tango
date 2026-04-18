# tango-cloud

Self-hosted cloud development platform for managing containerized projects, environments, and resources with git-based build pipelines, database backup/restore, DNS/domain management, and Traefik-based routing.

Monorepo: Go API + Vite React frontend, plus a dedicated backup runner for database dump/restore.

## Core Features

- Project & Environment Management
- Resource Lifecycle (database, app, service containers)
- Git-based Build Pipelines with BuildKit
- GitHub Source Connections (OAuth App + PAT)
- Domain Management (custom domains, base domains, wildcard routing)
- Database Backup & Restore for MySQL, MariaDB, PostgreSQL, and MongoDB
- Real-time Build & Run Logs via WebSocket
- Container Terminal (interactive shell over WebSocket)
- Multi-Channel Messaging (Discord, Telegram, Slack, WhatsApp)
- Encrypted Secrets & Environment Variables
- CLI with Docker health monitoring and auto-restart

## Structure

```
tango-cloud/
├── cmd/
│   ├── api/main.go            ← API server + embedded frontend
│   ├── backup-runner/main.go  ← stateless dump/restore runner
│   └── cli/main.go            ← CLI binary (daemon, service mgmt)
├── internal/
│   ├── auth/                  ← JWT, bcrypt, middleware
│   ├── application/           ← command/query handlers (CQRS)
│   ├── domain/                ← entities + repository interfaces
│   ├── infrastructure/        ← persistence, services, tools
│   ├── orchestrator/          ← pluggable driver interface (compose, k3s, swarm, nomad)
│   ├── runner/                ← backup runner HTTP + CLI execution
│   ├── channels/              ← messaging adapters
│   └── handler/               ← REST handlers + WebSocket
├── web/                       ← Vite + React + TanStack Router
├── docs/                      ← detailed documentation
├── Dockerfile                 ← app multi-stage build
├── Dockerfile.backup-runner   ← backup runner image
└── docker-compose.yml         ← production deploy
```

## Quick Start

### Requirements

- Go 1.22+
- Node.js 24+ and pnpm
- Docker + Docker Compose

### Local Development

```bash
# 1. Install dependencies
go mod tidy
cd web && pnpm install && cd ..

# 2. Start infra (Traefik, Postgres, BuildKit, backup-runner)
make infra

# 3. Run API + frontend dev server (each in its own terminal)
make dev        # loads .env.dev automatically → http://localhost:8080
make web-dev    # Vite dev server → http://localhost:5173 (proxies /api → :8080)
```

`make infra` creates `traefik/traefik.yml` and `letsencrypt/acme.json` on first run.
`make dev` loads `.env.dev` so no manual `export` is needed.

> **Note:** `.env.dev` is gitignored. Copy and adjust if you need custom values:
>
> ```bash
> cp .env.dev .env.dev.local  # optional personal overrides
> ```

#### Useful dev commands

| Command           | Description                                                             |
| ----------------- | ----------------------------------------------------------------------- |
| `make infra`      | Start dev infra containers (Traefik, Postgres, BuildKit, backup-runner) |
| `make infra-down` | Stop dev infra containers                                               |
| `make dev`        | Run API server locally with `.env.dev`                                  |
| `make web-dev`    | Run Vite frontend dev server                                            |
| `make test`       | Run Go tests                                                            |
| `make build`      | Build API binary to `bin/api`                                           |
| `make build-full` | Build frontend + embed into API binary                                  |

#### Testing Traefik locally

Traefik is included in the dev infra stack. With `TRAEFIK_CONFIG_DIR=./traefik/config` set in `.env.dev`, the app writes routing config files that Traefik picks up in real time (no restart needed).

To test domain routing locally, set `APP_DOMAIN` in `.env.dev` to a hostname that resolves to `127.0.0.1` (e.g. via `/etc/hosts`), then use the Settings UI to apply it.

HTTPS/Let's Encrypt cannot be tested locally — a publicly reachable domain is required.

### Production Deploy

```bash
# Basic install (HTTP only)
curl -fsSL https://raw.githubusercontent.com/OpenViCloud/tango/main/install.sh | sudo bash

# With admin credentials
sudo ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=yourpassword \
  bash -c "$(curl -fsSL https://raw.githubusercontent.com/OpenViCloud/tango/main/install.sh)"

# With HTTPS via Let's Encrypt + admin credentials
sudo ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=yourpassword \
  bash -s -- --email you@example.com --domain app.example.com --https \
  < <(curl -fsSL https://raw.githubusercontent.com/OpenViCloud/tango/main/install.sh)
```

`install.sh` installs Docker if missing, creates required directories, downloads `docker-compose.yml`, generates `traefik/traefik.yml`, writes `/opt/tango/.env`, installs the CLI daemon as a system service, and starts the full stack.

On first install, the script generates and stores these secrets in `/opt/tango/.env` with `root:root` ownership and `600` permissions:

- `POSTGRES_PASSWORD`
- `DATABASE_URL`
- `JWT_SECRET`
- `DATA_ENCRYPTION_KEY`

#### Admin account

Set `ADMIN_EMAIL` and `ADMIN_PASSWORD` before running `install.sh` to seed an admin account on first start. If not set, no account is created — you must add one manually via the DB or set the env vars and restart.

```bash
# Re-seed after changing credentials (only creates if email not already in DB)
sudo sh -c 'echo "ADMIN_EMAIL=admin@example.com" >> /opt/tango/.env'
sudo sh -c 'echo "ADMIN_PASSWORD=newpassword" >> /opt/tango/.env'
docker compose -f /opt/tango/docker-compose.yml restart app
```

After first login, change your password from **Settings → Account** and optionally create additional accounts or API keys.

After deployment, HTTPS settings (email, domain, TLS toggle) can also be changed at any time from the **Settings** page in the UI — the app rewrites `traefik/traefik.yml` and restarts Traefik automatically.

### Install CLI

```bash
curl -fsSL https://github.com/OpenViCloud/tango/releases/download/cli-latest/tango-linux-amd64 -o tango
chmod +x tango
sudo install -m 0755 tango /usr/local/bin/tango
```

See [CLI documentation](docs/cli.md) for orchestration commands, daemon setup, and service management.

## Documentation

| Doc                                    | Description                                                                 |
| -------------------------------------- | --------------------------------------------------------------------------- |
| [Architecture](docs/architecture.md)   | Backend conventions, DDD/CQRS structure, adding new modules                 |
| [CLI](docs/cli.md)                     | CLI commands, daemon health checks, service management, driver architecture |
| [Domain Model](docs/domain-model.md)   | Project/Resource hierarchy, build pipeline, backup/restore, routing         |
| [CI/CD](docs/ci-cd.md)                 | GitHub Actions workflow, Docker image builds, .dockerignore optimization    |
| [API Reference](docs/api-reference.md) | All REST endpoints and WebSocket paths                                      |
| [Configuration](docs/configuration.md) | All environment variables with defaults                                     |
| [Database](docs/database.md)           | GORM schema, tables, migration notes                                        |
| [Roadmap](docs/roadmap.md)             | Implementation phases and planned features                                  |
