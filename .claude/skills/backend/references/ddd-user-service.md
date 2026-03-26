# DDD User Service in Go

Use this reference when backend work should follow the repo's current DDD/CQRS/Onion direction without drifting away from the actual `tango` layout.

## Target Structure

```text
cmd/
└── api/main.go
internal/
├── contract/
│   └── common/paging.go       ← BaseRequestModel, BaseResponseModel
├── domain/
│   ├── user.go
│   └── user_repository.go
├── application/
│   ├── command/user.go
│   ├── query/user.go
│   └── services/              ← service interfaces/contracts only (no implementations)
├── infrastructure/
│   ├── db/
│   ├── persistence/
│   │   ├── models/
│   │   │   ├── user_record.go
│   │   │   └── models.go
│   │   └── repositories/
│   │       └── user_repository.go
│   └── services/              ← concrete service implementations
└── handler/
    └── rest/user_handler.go
```

## Dependency Flow

```text
interface -> application -> domain <- infrastructure
```

- `domain` depends on nobody
- `application` depends on domain contracts
- `infrastructure` implements domain contracts and infrastructure services
- `handler/rest` calls application handlers and maps DTOs

## Layer Responsibilities

### Domain

- Own entities, invariants, factories, and business behavior
- Define repository interfaces without knowing the database or ORM
- Set default values in constructors or factories such as `NewUser()`
- Keep transport concerns and persistence concerns out of the domain layer

### Application

- Separate writes from reads with CQRS
- Put mutations in `command/`
- Put reads in `query/`
- Coordinate domain objects and repository interfaces
- Avoid leaking ORM or HTTP details into use cases
- Define service interfaces in `application/services/` when the feature needs orchestration or provider-specific capabilities; put the concrete implementation in `infrastructure/services/`

### Infrastructure

- Hold DB setup, runtime integrations, repository implementations, and infrastructure-backed service implementations
- Implement domain repository interfaces
- Translate persistence models to and from domain entities
- Map duplicate-key and similar DB errors back to canonical domain errors

### Handler/REST

- Hold REST handlers, request DTOs, response DTOs, and HTTP concerns
- Call command or query handlers from the application layer
- Never return raw domain entities directly to clients
- Add Swagger annotations for public endpoints
- Prefer boundary logging here for request-scoped failures when no deeper audit/event stream exists

## HTTP API Mapping

| Method | Path | Use case |
| --- | --- | --- |
| POST | `/api/users` | `CreateUserCommand` |
| GET | `/api/users` | `ListUsersQuery` |
| GET | `/api/user/:id` | `GetUserByIDQuery` |
| GET | `/api/user/me` | current-user query |
| PUT | `/api/users/:id` | `UpdateUserCommand` |
| POST | `/api/users/:id/ban` | `BanUserCommand` |
| DELETE | `/api/users/:id` | `DeleteUserCommand` |

## Design Rules

- Set defaults in the domain, for example UUIDs and `CreatedAt` in `NewUser()`
- Validate on the write path, not while reading
- Read back after write operations when the caller needs the canonical stored state
- Use `FindByID` for nullable lookups and `GetByID` for required lookups that should error
- Prefer soft delete with domain behavior such as `SoftDelete()` and a `deleted_at` field
- Do not leak domain entities from handlers; map them to transport DTOs like `userResponse`
- Reuse `BaseRequestModel` and `BaseResponseModel` for paginated list endpoints
- Treat DB uniqueness as the last line of defense; application pre-checks and DB duplicate-key handling should both map to the same domain error
- Register every new persistence model in `internal/infrastructure/persistence/models/models.go`
- Wire new modules in `cmd/api/main.go`
- Regenerate Swagger docs after changing public REST contracts
- Use `log/slog` for backend logging, not `log.Printf`
- Request logging should live at middleware level with structured fields such as `traceId`, `method`, `path`, `status`, `latency_ms`, and `client_ip`
- Log expected business outcomes like duplicate-user conflicts as `info` or `warn`, not `error`
- Log system failures such as DB outages, provider failures, decrypt failures, and unexpected panics as `error`
- Avoid duplicate logs across repository, service, and handler layers; one primary log at the boundary is usually enough
- For third-party calls, log enough provider/request context to debug timeouts and failures without leaking secrets

## Adding a New Module

Use this checklist when adding a new DB-backed module such as `role`, `channel`, or `provider`:

1. Add the entity, errors, and list option/result types in `internal/domain/`
2. Add repository interfaces in `internal/domain/`
3. Add the GORM record in `internal/infrastructure/persistence/models/`
4. Register it in `models.go`
5. Implement repository translation and DB error mapping in `internal/infrastructure/persistence/repositories/`
6. Add command handlers for writes
7. Add query handlers for reads
8. Add an application service contract plus infrastructure implementation only if orchestration is needed
9. Add the REST handler with DTO mapping, route registration, and Swagger annotations
10. Wire the feature in `cmd/api/main.go`
11. Run `go test ./...`

## Quick Start

```bash
docker compose up -d postgres
go run ./cmd/api
```
