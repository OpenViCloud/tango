# tango-cloud

Monorepo: Go API + Vite React FE in a single Docker image.

Self-hosted cloud development platform for managing containerized projects, environments, and resources with git-based build pipelines, DNS/domain management, and Traefik-based routing.

## Core Features

- Project & Environment Management
- Resource Lifecycle (database, app, service containers)
- Git-based Build Pipelines with BuildKit
- GitHub Source Connections (OAuth App + PAT)
- Domain Management (custom domains, base domains, wildcard routing)
- Platform Settings for Traefik, TLS, and app domain exposure
- Real-time Build & Run Logs via WebSocket
- Container Terminal (interactive shell over WebSocket)
- Multi-Channel Messaging Integrations (Discord, Telegram, Slack, WhatsApp)
- Encrypted Secrets & Environment Variables
- Simple Self-Hosting

## Structure

```
tango-cloud/
├── cmd/
│   └── api/main.go            ← API server + FE serving + SPA fallback
├── internal/
│   ├── auth/                  ← JWT, bcrypt, middleware
│   ├── application/           ← command/query handlers
│   ├── contract/              ← shared request/response contracts
│   ├── domain/                ← entities + repository interfaces
│   ├── config/                ← shared config
│   ├── infrastructure/        ← DB bootstrap, persistence, server runtime
│   ├── channels/              ← messaging transport adapters (Discord, Slack, ...)
│   └── handler/               ← HTTP handlers + WebSocket
├── web/                       ← Vite + React + TanStack Router
├── Dockerfile                 ← multi-stage build
├── docker-compose.yml         ← app + postgres + traefik
├── Makefile
└── install.sh
```

## Backend Conventions

The backend follows a pragmatic DDD/CQRS split:

- `internal/domain/`
  Entities, domain errors, list options/results, repository interfaces
- `internal/application/command/`
  Write use cases: create, update, delete, start, stop
- `internal/application/query/`
  Read use cases: get by id, list
- `internal/application/services/`
  Application-facing service contracts for orchestration and runtime features
- `internal/infrastructure/persistence/models/`
  GORM persistence records used by `AutoMigrate()`
- `internal/infrastructure/persistence/repositories/`
  Repository implementations and DB error mapping
- `internal/infrastructure/services/`
  Service implementations: Docker runtime, BuildKit, GitHub integration, channel runtimes
- `internal/handler/rest/`
  HTTP handlers, request DTOs, response DTOs, error mapping
- `cmd/api/main.go`
  Composition root: instantiate repositories/services/handlers and register routes

### Adding a new backend module

When adding a new DB-backed module, follow this order:

1. Add the domain entity and domain errors in `internal/domain/`
2. Add the repository interface in `internal/domain/`
3. Add the GORM record in `internal/infrastructure/persistence/models/`
4. Register the new record in `internal/infrastructure/persistence/models/models.go`
5. Implement the repository in `internal/infrastructure/persistence/repositories/`
6. Add write use cases in `internal/application/command/` and read use cases in `internal/application/query/`
7. If the feature needs provider-specific or runtime logic, add an application service contract in `internal/application/services/` and the implementation in `internal/infrastructure/services/`
8. Add the REST handler in `internal/handler/rest/`
9. Wire everything in `cmd/api/main.go`
10. Add Swagger annotations and regenerate `docs/`
11. Run `go test ./...`

### Rules of thumb

- Put validation and invariants on the write path in domain constructors or command handlers
- Do not return raw domain entities from REST handlers; map them to transport DTOs
- Reuse `internal/contract/common.BaseRequestModel` and `BaseResponseModel` for paginated list endpoints
- Use one canonical domain error per business case, then map it to one public API error code
- Keep DB unique constraints as the final line of defense and map duplicate-key errors back to the same domain error
- Use `FindBy...` for nullable lookups and `GetBy...` for required lookups
- Add Swagger comments for every public REST endpoint and regenerate docs after changing the API

## Domain Model

### Project → Environment → Resource

The core hierarchy:

- **Project** — top-level grouping (e.g. "my-saas-app")
- **Environment** — subdivision within a project (e.g. "development", "staging"); environments can be forked
- **Resource** — a single containerized unit inside an environment, one of three types:
  - `db` — database (Postgres, MySQL, Redis, MongoDB, etc.)
  - `app` — application container built from a git repo or Docker image
  - `service` — supporting service container

### Build Pipeline

Resources of type `app` can be built from source:

1. User connects a GitHub repo via OAuth App or PAT (`source_connections`)
2. User selects repo + branch on the resource
3. A `build_job` is created — clones the repo, builds the Docker image with BuildKit, pushes to registry
4. On build completion, the resource starts automatically
5. Build logs stream in real-time via WebSocket

### Resource Lifecycle

```
queued → pulling_image → creating → starting → running
                                              ↓
                                           stopped / error
```

### Domain Routing

Resources can expose HTTP services through managed domains:

- **Base domains** — platform-managed domains such as `apps.example.com`
- **Wildcard mode** — allow generated subdomains like `api.apps.example.com`
- **Custom domains** — user-managed domains verified by DNS
- **TLS routing** — per-domain HTTP/HTTPS behavior via Traefik labels and file provider config

Platform-level routing is configured from the Settings page and persisted in platform config tables.

### Channel Integrations

Channels connect external messaging platforms to the platform:

- Discord bot runtime
- Telegram bot runtime
- Slack workspace integration
- WhatsApp integration (QR code provisioning)

Credentials are encrypted at rest.

## Requirements

- Go 1.22+
- Node.js 20+ and pnpm
- Docker + Docker Compose
- BuildKit daemon (for build pipeline)
- Public DNS / server IP if using custom domains and wildcard routing

## Run Locally

### 1. Install Go

```bash
brew install go
go version  # go version go1.22.x darwin/arm64
```

### 2. Install dependencies

```bash
go mod tidy

cd web && pnpm install && cd ..
```

### 3. Set environment variables

```bash
export JWT_SECRET=mysecretkey123
export PORT=8080
export DB_DRIVER=postgres
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/tango_cloud?sslmode=disable
export LLM_CONFIG_ENCRYPTION_KEY=12345678901234567890123456789012
```

Local SQLite:

```bash
export DB_DRIVER=sqlite
export DATABASE_URL='file:tango_cloud.db?_foreign_keys=on'
```

BuildKit (required for git-based builds):

```bash
docker run -d \
  --name buildkitd \
  --privileged \
  -p 1234:1234 \
  moby/buildkit:latest \
  --addr tcp://0.0.0.0:1234

export BUILDKIT_HOST=tcp://localhost:1234
export BUILD_WORKSPACE_DIR=/tmp/tango-builds
```

Traefik / domain routing (required only for domain exposure):

```bash
export TRAEFIK_NETWORK=bridge
export APP_DOMAIN=app.example.com
export APP_TLS_ENABLED=true
export APP_BACKEND_URL=http://app:8080
export RESOURCE_MOUNT_ROOT=/absolute/host/path/to/tango-cloud/data/resource-volumes
```

Optional channel integrations:

```bash
# Discord
export DISCORD_BOT_TOKEN='your_discord_bot_token'

# Telegram
export TELEGRAM_BOT_TOKEN='your_telegram_bot_token'
```

Notes:

- `LLM_CONFIG_ENCRYPTION_KEY` must be exactly 32 characters long.
- `BUILDKIT_HOST` is required only if using git-based resource builds.
- `PUBLIC_IP`, base domains, and wildcard DNS are required only if using custom/base-domain routing.
- `RESOURCE_MOUNT_ROOT` must be an absolute host path visible to the Docker daemon. Tango resolves resource mounts under this root and rejects absolute source paths from resource config.

### Resource volume mounts

Resource volume mounts are scoped to one shared host root instead of arbitrary host paths.

- Default host root: `/tmp/tango-resource-volumes`
- Default app-visible path inside the `app` container: `/platform/resource-volumes`
- `install.sh` overrides the host root to `<repo>/data/resource-volumes` and creates that directory automatically
- Resource volume entries must use `source:target[:mode]`, where `source` is a relative subpath under `RESOURCE_MOUNT_ROOT`

Examples:

```text
databasus-data:/databasus-data
project-a/uploads:/app/uploads
shared/cache:/data:ro
```

Rules:

- Source must be relative and cannot escape the configured root with `..`
- Target must be an absolute container path
- Only `ro` and `rw` modes are accepted
- Tango creates the source directory automatically before starting the resource

### 4. Run BE + FE in parallel

```bash
# Terminal 1 — API server
go run ./cmd/api
# → http://localhost:8080

# Terminal 2 — FE dev server
cd web && pnpm dev
# → http://localhost:5173 (proxy /api → :8080)
```

## Build And Run With Docker

```bash
docker compose up --build

# http://localhost:8080 → Web UI + API
# demo login:
# email: demo.admin@example.com
# password: password123
```

`docker-compose.yml` mounts `${RESOURCE_MOUNT_ROOT}` into the `app` container and passes the same host path to the API so new resources can bind subdirectories under that root.

## API Endpoints

### Auth

| Method | Path                 | Auth | Description    |
| ------ | -------------------- | ---- | -------------- |
| POST   | /api/auth/login      | ❌   | Log in         |
| POST   | /api/auth/refresh    | ❌   | Refresh token  |
| POST   | /api/auth/logout     | ❌   | Log out        |
| GET    | /api/user/me         | ✅   | Current user   |

### Projects & Environments

| Method | Path                                  | Auth | Description              |
| ------ | ------------------------------------- | ---- | ------------------------ |
| GET    | /api/projects                         | ✅   | List projects            |
| POST   | /api/projects                         | ✅   | Create project           |
| GET    | /api/projects/:id                     | ✅   | Get project              |
| PUT    | /api/projects/:id                     | ✅   | Update project           |
| DELETE | /api/projects/:id                     | ✅   | Delete project           |
| POST   | /api/projects/:id/environments        | ✅   | Add environment          |
| POST   | /api/environments/:envId/fork         | ✅   | Fork environment         |

### Resources

| Method | Path                                          | Auth | Description                  |
| ------ | --------------------------------------------- | ---- | ---------------------------- |
| GET    | /api/environments/:envId/resources            | ✅   | List resources in env        |
| POST   | /api/environments/:envId/resources            | ✅   | Create resource (from image) |
| POST   | /api/environments/:envId/resources/from-git   | ✅   | Create resource from git     |
| GET    | /api/resources/:id                            | ✅   | Get resource                 |
| PUT    | /api/resources/:id                            | ✅   | Update resource              |
| DELETE | /api/resources/:id                            | ✅   | Delete resource              |
| POST   | /api/resources/:id/start                      | ✅   | Start resource               |
| POST   | /api/resources/:id/stop                       | ✅   | Stop resource                |
| POST   | /api/resources/:id/build                      | ✅   | Trigger build                |
| GET    | /api/resources/:id/logs                       | ✅   | Get run logs                 |
| GET    | /api/resources/:id/env-vars                   | ✅   | List env vars                |
| PUT    | /api/resources/:id/env-vars                   | ✅   | Update env vars              |

### Routing & Settings

| Method | Path                               | Auth | Description                         |
| ------ | ---------------------------------- | ---- | ----------------------------------- |
| GET    | /api/settings                      | ✅   | Get platform settings               |
| PATCH  | /api/settings                      | ✅   | Update platform settings            |
| GET    | /api/settings/base-domains         | ✅   | List managed base domains           |
| POST   | /api/settings/base-domains         | ✅   | Add base domain                     |
| DELETE | /api/settings/base-domains/:id     | ✅   | Delete base domain                  |
| GET    | /api/domains/check                 | ✅   | Check whether a hostname is in use  |

### Builds

| Method | Path                  | Auth | Description              |
| ------ | --------------------- | ---- | ------------------------ |
| GET    | /api/builds           | ✅   | List build jobs          |
| POST   | /api/builds           | ✅   | Create build from git    |
| POST   | /api/builds/upload    | ✅   | Build from archive upload|
| GET    | /api/builds/:id       | ✅   | Get build job            |
| POST   | /api/builds/:id/cancel| ✅   | Cancel build             |

### Source Connections (GitHub)

| Method | Path                                                     | Auth | Description              |
| ------ | -------------------------------------------------------- | ---- | ------------------------ |
| POST   | /api/source-connections/github/apps                      | ✅   | Begin GitHub OAuth flow  |
| POST   | /api/source-connections/pat                              | ✅   | Add PAT connection       |
| GET    | /api/source-connections                                  | ✅   | List connections         |
| DELETE | /api/source-connections/:id                              | ✅   | Remove connection        |
| GET    | /api/source-connections/:id/repos                        | ✅   | List repos               |
| GET    | /api/source-connections/:id/repos/:owner/:repo/branches  | ✅   | List branches            |

### Channels

| Method | Path              | Auth | Description              |
| ------ | ----------------- | ---- | ------------------------ |
| GET    | /api/channels     | ✅   | List channels            |
| POST   | /api/channels     | ✅   | Create channel           |
| GET    | /api/channels/:id | ✅   | Get channel              |
| DELETE | /api/channels/:id | ✅   | Delete channel           |

### WebSocket

| Path                              | Description                      |
| --------------------------------- | -------------------------------- |
| /api/ws/builds/:id                | Stream build logs                |
| /api/ws/resource-runs/:id         | Stream resource run logs         |
| /api/ws/resources/:id/terminal    | Interactive container shell      |

## Database

The app uses GORM with `AutoMigrate()` on boot.

Select the runtime DB with `DB_DRIVER=postgres|sqlite`.

Tables managed by GORM:

- `users`
- `roles`
- `user_roles`
- `projects`
- `environments`
- `resources`
- `resource_ports`
- `resource_env_vars`
- `resource_runs`
- `build_jobs`
- `channels`
- `source_providers`
- `source_connections`

Notes:

- Simple schema changes (new fields) can rely on `AutoMigrate()`.
- For breaking schema changes or data migrations, write a dedicated migration script.
- When adding a new schema, register the GORM record in `internal/infrastructure/persistence/models/models.go`.
