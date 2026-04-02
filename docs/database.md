# Database

The app uses GORM with `AutoMigrate()` on boot.

Select the runtime DB with `DB_DRIVER=postgres|sqlite`.

## Tables managed by GORM

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
- `database_sources`
- `storages`
- `backup_configs`
- `backups`
- `restores`
- `channels`
- `source_providers`
- `source_connections`

## Entity Relationship Diagram

```mermaid
erDiagram

  %% IDENTITY
  users {
    varchar id PK
    text email
    text nickname
    text first_name
    text last_name
    text password_hash
    text status
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  roles {
    text id PK
    text name
    text description
    bool is_system
    timestamptz created_at
    timestamptz updated_at
  }
  user_roles {
    text user_id FK
    text role_id FK
    timestamptz created_at
  }

  %% PROJECTS
  projects {
    varchar id PK
    varchar name
    varchar description
    varchar status
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  environments {
    varchar id PK
    varchar project_id FK
    varchar name
    varchar status
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }

  %% RESOURCES
  resources {
    varchar id PK
    varchar environment_id FK
    varchar name
    varchar type "db | app | service"
    varchar status
    varchar image
    text config_json
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  resource_domains {
    varchar id PK
    varchar resource_id FK
    varchar host
    varchar type "auto | custom"
    bool tls_enabled
    bool verified
    timestamptz verified_at
    timestamptz created_at
    timestamptz updated_at
  }
  resource_ports {
    varchar id PK
    varchar resource_id FK
    int host_port
    int internal_port
    varchar protocol
    timestamptz created_at
    timestamptz updated_at
  }
  platform_configs {
    varchar key PK
    text value
    timestamptz updated_at
  }
  base_domains {
    varchar id PK
    varchar domain
    bool wildcard_enabled
    timestamptz created_at
    timestamptz updated_at
  }
  resource_env_vars {
    varchar id PK
    varchar resource_id FK
    varchar key
    text encrypted_value
    bool is_secret
    timestamptz created_at
    timestamptz updated_at
  }
  resource_runs {
    varchar id PK
    varchar resource_id FK
    varchar status
    varchar container_id
    timestamptz started_at
    timestamptz stopped_at
    timestamptz created_at
    timestamptz updated_at
  }

  %% BACKUPS
  database_sources {
    varchar id PK
    varchar resource_id FK
    varchar name
    varchar db_type "postgres | mysql | mariadb | mongodb"
    varchar host
    int port
    varchar username
    text password_encrypted
    varchar database_name
    varchar version
    bool is_tls_enabled
    varchar auth_database
    text connection_uri_encrypted
    timestamptz created_at
    timestamptz updated_at
  }
  storages {
    varchar id PK
    varchar name
    varchar type "local | s3 | minio"
    text config_json
    text credentials_encrypted
    timestamptz created_at
    timestamptz updated_at
  }
  backup_configs {
    varchar id PK
    varchar database_source_id FK
    varchar storage_id FK
    bool is_enabled
    varchar schedule_type
    varchar time_of_day
    int interval_hours
    varchar retention_type
    int retention_days
    int retention_count
    bool is_retry_if_failed
    int max_retry_count
    varchar encryption_type
    varchar compression_type
    varchar backup_method
    timestamptz created_at
    timestamptz updated_at
  }
  backups {
    varchar id PK
    varchar database_source_id FK
    varchar backup_config_id FK
    varchar storage_id FK
    varchar status
    varchar backup_method
    varchar file_name
    text file_path
    bigint file_size_bytes
    varchar checksum_sha256
    timestamptz started_at
    timestamptz completed_at
    bigint duration_ms
    text fail_message
    varchar encryption_type
    text metadata_json
    timestamptz created_at
  }
  restores {
    varchar id PK
    varchar backup_id FK
    varchar database_source_id FK
    varchar status
    varchar target_host
    int target_port
    varchar target_username
    text target_password_encrypted
    varchar target_database_name
    varchar target_auth_database
    text target_uri_encrypted
    timestamptz started_at
    timestamptz completed_at
    bigint duration_ms
    text fail_message
    text metadata_json
    timestamptz created_at
  }

  %% BUILDS
  build_jobs {
    varchar id PK
    varchar resource_id FK
    varchar status "pending | running | succeeded | failed | cancelled"
    varchar repo_url
    varchar branch
    varchar image_tag
    text logs
    timestamptz started_at
    timestamptz finished_at
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }

  %% SOURCE CONNECTIONS
  source_providers {
    varchar id PK
    varchar kind "github_app"
    text encrypted_credentials
    timestamptz created_at
    timestamptz updated_at
  }
  source_connections {
    varchar id PK
    varchar user_id FK
    varchar provider_id FK
    varchar kind "github_app | pat"
    text encrypted_credentials
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }

  %% CHANNELS
  channels {
    varchar id PK
    varchar name
    varchar kind "discord | telegram | slack | whatsapp"
    varchar status
    text encrypted_credentials
    text settings_json
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }

  %% RELATIONSHIPS
  users ||--o{ user_roles : ""
  roles ||--o{ user_roles : ""

  projects ||--o{ environments : ""
  environments ||--o{ resources : ""
  resources ||--o{ resource_domains : ""
  resources ||--o{ resource_ports : ""
  resources ||--o{ resource_env_vars : ""
  resources ||--o{ resource_runs : ""
  resources ||--o{ build_jobs : ""
  resources ||--o{ database_sources : ""

  database_sources ||--o{ backup_configs : ""
  storages ||--o{ backup_configs : ""
  database_sources ||--o{ backups : ""
  storages ||--o{ backups : ""
  backups ||--o{ restores : ""
  database_sources ||--o{ restores : ""

  users ||--o{ source_connections : ""
  source_providers ||--o{ source_connections : ""
```

## Notes

- Simple schema changes (new fields) can rely on `AutoMigrate()`.
- For breaking schema changes or data migrations, write a dedicated migration script.
- When adding a new schema, register the GORM record in `internal/infrastructure/persistence/models/models.go`.
