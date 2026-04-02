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

## Structure

```
tango-cloud/
├── cmd/
│   ├── api/main.go            ← API server + embedded frontend
│   └── backup-runner/main.go  ← stateless dump/restore runner
├── internal/
│   ├── auth/                  ← JWT, bcrypt, middleware
│   ├── application/           ← command/query handlers (CQRS)
│   ├── domain/                ← entities + repository interfaces
│   ├── infrastructure/        ← persistence, services, tools
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
- Node.js 20+ and pnpm
- Docker + Docker Compose

### Run locally

```bash
# 1. Install dependencies
go mod tidy
cd web && pnpm install && cd ..

# 2. Set minimum env vars
export JWT_SECRET=mysecretkey123
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/tango?sslmode=disable'
export LLM_CONFIG_ENCRYPTION_KEY=12345678901234567890123456789012

# 3. Run API + frontend dev server
go run ./cmd/api          # http://localhost:8080
cd web && pnpm dev        # http://localhost:5173 (proxy /api → :8080)
```

### Run with Docker

```bash
docker compose up --build
# http://localhost:8080
# demo: demo.admin@example.com / password123
```

## Documentation

| Doc | Description |
| --- | ----------- |
| [Architecture](docs/architecture.md) | Backend conventions, DDD/CQRS structure, adding new modules |
| [Domain Model](docs/domain-model.md) | Project/Resource hierarchy, build pipeline, backup/restore, routing |
| [CI/CD](docs/ci-cd.md) | GitHub Actions workflow, Docker image builds, .dockerignore optimization |
| [API Reference](docs/api-reference.md) | All REST endpoints and WebSocket paths |
| [Configuration](docs/configuration.md) | All environment variables with defaults |
| [Database](docs/database.md) | GORM schema, tables, migration notes |
| [Roadmap](docs/roadmap.md) | Implementation phases and planned features |
