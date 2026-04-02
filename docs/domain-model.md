# Domain Model

## Hierarchy

```
Project
└── Environment (fork-able)
    └── Resource (db | app | service)
        ├── ResourcePort
        ├── ResourceEnvVar
        ├── ResourceDomain
        └── ResourceRun

PlatformConfig
└── BaseDomain

DatabaseSource
├── BackupConfig
├── Backup
└── Restore

Storage
└── BackupConfig
```

## Resource Types

| Type      | Description                                                   |
| --------- | ------------------------------------------------------------- |
| `db`      | Database containers (Postgres, MySQL, MariaDB, Redis, MongoDB, etc.) |
| `app`     | Application containers; can be built from a git repository    |
| `service` | Supporting service containers                                 |

## Resource Lifecycle

```mermaid
stateDiagram-v2
    [*] --> queued: created
    queued --> pulling_image: start
    pulling_image --> creating: image ready
    creating --> starting: container created
    starting --> running: container started
    running --> stopped: stop
    running --> error: failure
    stopped --> starting: restart
    error --> [*]
```

## Build Pipeline

Resources of type `app` can be built from source:

```mermaid
sequenceDiagram
    User->>API: POST /resources/from-git (repo + branch)
    API->>BuildService: CreateBuildJob(repo, branch, resourceID)
    BuildService->>GitHub: Clone repository
    BuildService->>BuildKit: Build Docker image
    BuildKit-->>BuildService: Stream build logs
    BuildService->>Registry: Push image
    BuildService->>ResourceService: StartResource(resourceID, image)
    ResourceService->>Docker: Create + start container
    Docker-->>ResourceService: Container running
```

### Build Job Lifecycle

```
pending → running → succeeded
                 → failed
                 → cancelled
```

Real-time build logs are streamed to the browser via WebSocket at `/api/ws/builds/:id`.

## Source Connections

Source connections store credentials for accessing private git repositories.

| Type         | Description                                      |
| ------------ | ------------------------------------------------ |
| `github_app` | GitHub OAuth App (manifest flow, broader scope)  |
| `pat`        | Personal Access Token (simpler, user-scoped)     |

Credentials (tokens, keys) are AES-encrypted before being stored in the database.

```mermaid
sequenceDiagram
    User->>API: POST /source-connections/github/apps
    API->>GitHub: Create App via manifest
    GitHub-->>API: App credentials + installation token
    API->>DB: Store encrypted credentials
    User->>API: GET /source-connections/:id/repos
    API->>GitHub: List repositories
    GitHub-->>API: Repo list
    API-->>User: Repository list
```

## Database Backup & Restore

Database backup/restore is split into two responsibilities:

1. **`cmd/api`** — stores backup sources, storages, backup configs, backups, restores; exposes REST endpoints and UI; orchestrates backup and restore jobs

2. **`cmd/backup-runner`** — stateless internal service; runs database CLI tools; detects MySQL/MariaDB/PostgreSQL versions when needed; streams dump/restore data back to the API

### Supported databases

- MySQL logical dump / restore via `mysqldump` / `mysql`
- MariaDB logical dump / restore via `mariadb-dump` / `mariadb`
- PostgreSQL logical dump / restore via `pg_dump -Fc` / `pg_restore`
- MongoDB logical dump / restore via `mongodump --archive` / `mongorestore --archive`
- Local storage backend (`none` or `gzip` compression)

The API persists metadata and artifact references. The runner does not keep its own database.

### Backup Execution Flow

```mermaid
sequenceDiagram
    User->>API: POST /api/backup-sources/:id/backups
    API->>DB: Create backup record (pending)
    API->>BackupExecutor: ExecuteBackup(backupID)
    BackupExecutor->>BackupRunner: POST /internal/<db>/logical-dump
    BackupRunner->>Database: mysqldump / mariadb-dump / pg_dump / mongodump
    BackupRunner-->>BackupExecutor: stream artifact bytes
    BackupExecutor->>StorageDriver: store local artifact
    BackupExecutor->>DB: Update backup record (completed/failed)
```

### Restore Execution Flow

```mermaid
sequenceDiagram
    User->>API: POST /api/backups/:id/restore
    API->>DB: Create restore record (pending)
    API->>RestoreExecutor: ExecuteRestore(restoreID)
    RestoreExecutor->>StorageDriver: load artifact to temp path
    RestoreExecutor->>BackupRunner: POST /internal/<db>/logical-restore
    BackupRunner->>Database: mysql / mariadb / pg_restore / mongorestore
    BackupRunner-->>RestoreExecutor: status
    RestoreExecutor->>DB: Update restore record (completed/failed)
```

## Domain Routing

Resources can expose HTTP services through managed domains:

| Type     | Description |
| -------- | ----------- |
| `auto`   | Generated from a managed base domain, optionally using wildcard DNS |
| `custom` | User-supplied hostname that must be DNS-verified before secure routing |

Platform settings control: public IP, app domain, app TLS, Traefik Docker network, certificate resolver, managed base domains.

```mermaid
sequenceDiagram
    User->>API: Add base domain / custom domain
    API->>DB: Persist PlatformConfig / BaseDomain / ResourceDomain
    User->>API: Start or restart resource
    API->>ResourceRunService: Resolve routing config
    ResourceRunService->>Traefik: Generate labels / file-provider config
    ResourceRunService->>Docker: Create or update container
    Traefik-->>Internet: Route hostnames to the resource
```

## Channel Integrations

Channels connect external messaging platforms to the platform:

| Kind       | Description                             |
| ---------- | --------------------------------------- |
| `discord`  | Discord bot integration                 |
| `telegram` | Telegram bot integration                |
| `slack`    | Slack workspace integration             |
| `whatsapp` | WhatsApp (QR code provisioning)         |

Channel credentials are encrypted at rest. Each channel runs an independent runtime goroutine started/stopped via the channel service.
