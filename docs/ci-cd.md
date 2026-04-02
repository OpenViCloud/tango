# CI/CD

The project uses GitHub Actions (`.github/workflows/build.yml`) to build and push multi-arch Docker images to Docker Hub. There are two images with different build strategies:

| Image | Registry | Trigger | Frequency |
| ----- | -------- | ------- | --------- |
| `timegroups/tango-cloud` | Docker Hub | Push a git tag | Every release |
| `timegroups/tango-backup-runner` | Docker Hub | Manual trigger only | Rarely (when DB tools or runner code changes) |

Both images are built for `linux/amd64` and `linux/arm64`.

## Build the app (automatic on release)

Create and push a tag to trigger the app build:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This builds `Dockerfile`, tags the image as `timegroups/tango-cloud:v1.0.0` and `timegroups/tango-cloud:latest`, and pushes to Docker Hub.

## Build the backup runner (manual)

Go to **Actions → Build → Run workflow**, tick **"Build backup-runner"**, and run. This builds `Dockerfile.backup-runner` and pushes `timegroups/tango-backup-runner:latest` to Docker Hub.

Only trigger this when:
- Runner Go code changes (`cmd/backup-runner/`, `internal/runner/`, `internal/infrastructure/tools/`)
- DB tool binaries in `assets/tools/` are updated
- `Dockerfile.backup-runner` is modified

## Required GitHub secrets

| Secret | Value |
| ------ | ----- |
| `DOCKERHUB_USERNAME` | Docker Hub username |
| `DOCKERHUB_TOKEN` | Docker Hub Access Token (Read & Write scope) |

Set these in **Repo Settings → Secrets and variables → Actions**.

## Docker build context optimization

Each Dockerfile has a dedicated `.dockerignore` file (requires BuildKit):

| File | Used by | Key exclusions |
| ---- | ------- | -------------- |
| `Dockerfile.dockerignore` | App build | `assets/tools/` (210MB), `web/node_modules/` |
| `Dockerfile.backup-runner.dockerignore` | Runner build | `web/` (487MB), `migrations/` |

This reduces build context from ~700MB to ~10MB (app) and ~170MB (runner).

## Build locally (optional)

App image:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t timegroups/tango-cloud:latest \
  --push .
```

Backup runner image:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f Dockerfile.backup-runner \
  -t timegroups/tango-backup-runner:latest \
  --push .
```

## Backup runner image contents

`Dockerfile.backup-runner` bundles all database CLI tools the runner needs:

- MySQL client binaries from `assets/tools/` → `/usr/local/`
- MariaDB client binaries from `assets/tools/` → `/usr/local/mariadb/`
- PostgreSQL client binaries (versions 12–18) from `assets/tools/` → `/usr/lib/postgresql/<version>/bin/`
- MongoDB Database Tools (downloaded during build) → `/usr/local/mongodb-database-tools/bin/`

For deploy, `docker-compose.yml` points `backup-runner` to:

```text
BACKUP_RUNNER_BASE_URL=http://backup-runner:8081
MYSQL_INSTALL_DIR=/usr/local
MARIADB_INSTALL_DIR=/usr/local/mariadb
POSTGRES_INSTALL_DIR=/usr/lib/postgresql
MONGODB_TOOLS_DIR=/usr/local/mongodb-database-tools
```
