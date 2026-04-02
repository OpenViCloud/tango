# Configuration

All configuration is via environment variables. The app reads them on startup.

## Required

| Variable | Description | Example |
| -------- | ----------- | ------- |
| `JWT_SECRET` | Secret key for JWT signing | `mysecretkey123` |
| `DATABASE_URL` | Database connection string | `postgres://postgres:postgres@localhost:5432/tango?sslmode=disable` |
| `LLM_CONFIG_ENCRYPTION_KEY` | 32-character encryption key | `12345678901234567890123456789012` |

## Core

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `PORT` | `8080` | API server port |
| `DB_DRIVER` | `postgres` | `postgres` or `sqlite` |
| `DATABASE_URL` | (postgres default) | Postgres or SQLite connection string |
| `API_BASE_URL` | `http://localhost:8080` | Public API base URL |
| `FRONTEND_BASE_URL` | same as `API_BASE_URL` | Public frontend URL |
| `CACHE_DRIVER` | `memory` | Cache backend |
| `CACHE_DEFAULT_TTL` | `1m` | Default cache TTL |

For SQLite:

```bash
export DB_DRIVER=sqlite
export DATABASE_URL='file:tango.db?_foreign_keys=on'
```

## Build Pipeline

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `BUILDKIT_HOST` | `tcp://buildkitd:1234` | BuildKit daemon address |
| `BUILD_WORKSPACE_DIR` | `/tmp/tango-builds` | Workspace for build jobs |
| `BUILD_REGISTRY_HOST` | (empty) | Docker registry host |
| `BUILD_REGISTRY_USER` | (empty) | Registry username |
| `BUILD_REGISTRY_PASS` | (empty) | Registry password |

## Backup Runner

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `BACKUP_RUNNER_BASE_URL` | `http://127.0.0.1:8081` | Runner HTTP endpoint |
| `BACKUP_RUNNER_TOKEN` | (empty) | Bearer token for runner auth |
| `BACKUP_RUNNER_PORT` | `8081` | Runner listen port |
| `MYSQL_INSTALL_DIR` | auto-detected | MySQL tools directory |
| `MARIADB_INSTALL_DIR` | auto-detected | MariaDB tools directory |
| `POSTGRES_INSTALL_DIR` | auto-detected | PostgreSQL tools directory |
| `MONGODB_TOOLS_DIR` | `/usr/local/mongodb-database-tools` | MongoDB tools directory |

## Domain Routing

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `TRAEFIK_DOCKER_NETWORK` | `tango_net` | Docker network for Traefik |
| `TRAEFIK_CONFIG_DIR` | (empty) | Traefik file provider config directory |
| `APP_DOMAIN` | (empty) | Platform domain |
| `APP_TLS_ENABLED` | `false` | Enable TLS |
| `APP_BACKEND_URL` | `http://app:8080` | Backend URL for Traefik routing |
| `PUBLIC_IP` | (empty) | Server public IP |

## Resource Volumes

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `RESOURCE_MOUNT_ROOT` | `/tmp/tango-resource-volumes` | Host path for resource volumes |
| `RESOURCE_MOUNT_ROOT_APP` | `/platform/resource-volumes` | App-visible path inside container |

Resource volume mounts are scoped to one shared host root instead of arbitrary host paths.

- `install.sh` overrides the host root to `<repo>/data/resource-volumes`
- Resource volume entries must use `source:target[:mode]`, where `source` is a relative subpath
- Source must be relative and cannot escape the root with `..`
- Target must be an absolute container path
- Only `ro` and `rw` modes are accepted

Examples:

```text
databasus-data:/databasus-data
project-a/uploads:/app/uploads
shared/cache:/data:ro
```

## Channel Integrations

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `DISCORD_BOT_TOKEN` | (empty) | Discord bot token |
| `DISCORD_REQUIRE_MENTION` | `true` | Require @mention |
| `DISCORD_ENABLE_TYPING` | `true` | Show typing indicator |
| `DISCORD_ALLOWED_USER_IDS` | (empty) | Comma-separated allowed user IDs |
| `TELEGRAM_BOT_TOKEN` | (empty) | Telegram bot token |
| `TELEGRAM_ENABLE_TYPING` | `true` | Show typing indicator |
| `TELEGRAM_ALLOWED_USER_IDS` | (empty) | Comma-separated allowed user IDs |

## Logging

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `LOG_FORMAT` | `text` | `text` or `json` |
| `LOG_OUTPUT` | `both` | `stdout`, `file`, or `both` |
| `LOG_FILE_PATH` | `logs/tango.log` | Log file path |
| `LOG_MAX_SIZE_MB` | `20` | Max log file size before rotation |
| `LOG_MAX_BACKUPS` | `10` | Max rotated log files to keep |
| `LOG_MAX_AGE_DAYS` | `7` | Max age of rotated log files |
| `LOG_COMPRESS` | `true` | Compress rotated logs |
