---
name: backend
description: "Use when the task is about the backend in this repository: Go service structure, DDD + CQRS + Onion Architecture, files under cmd/ or internal/, API routes and handlers, domain and repository boundaries, auth, config, Docker build, or frontend asset embedding."
---

# Backend

## Overview

This skill covers the Go backend in this repository. Use it for service structure, API behavior, domain modeling, repository implementations, auth, config, server-side integration, and Docker packaging.

## Project Scope

- Current repo entrypoints: `cmd/api/main.go`, `cmd/backup-runner/main.go`, `cmd/cli/main.go`
- Current shared packages: `internal/auth`, `internal/contract`, `internal/config`, `internal/channels`, `internal/messaging`, `internal/runner`
- Target service pattern for backend feature work: DDD + CQRS + Onion Architecture
- Reference architecture: `references/ddd-user-service.md`
- Current HTTP/framework/runtime details must be verified from code before editing

## Working Rules

1. Read the exact handler, middleware, or package touched by the task before making changes.
2. For new backend service design, prefer the DDD/CQRS/Onion split documented in `references/ddd-user-service.md`.
3. Keep dependency direction explicit: `interface -> application -> domain <- infrastructure`.
4. Do not let `domain` depend on HTTP, DB, ORM, ent, Echo, Gin, or transport DTOs.
5. Put validation and invariants on the write path, typically in domain factories or command handlers, not in read models.
6. When changing auth or API contracts, trace both server and client impact before landing changes.
7. If changing config or env vars, check code, docs, and Docker files for drift.
8. Prefer the repo's actual backend layout over generic examples when they conflict with older docs.
9. Public REST endpoints should include Swagger annotations and regenerated `docs/` when the contract changes.
10. Reuse one canonical domain error per business case and map duplicate DB errors back to that same domain error.
11. Use `log/slog` for backend logging. Do not introduce new `log.Printf` style logging in backend code.

## Architecture Guidance

- Read `references/ddd-user-service.md` when the task involves:
  - introducing a new backend module or service
  - reorganizing handlers, domain models, repositories, or use cases
  - separating commands from queries
  - deciding where validation, soft delete, or DTO mapping should live
- For DB-backed user work, prefer this shape:
  - `internal/contract/` for shared request/response contracts
  - `internal/domain/` for entities and repository interfaces
  - `internal/application/command` and `internal/application/query` for CQRS use cases
  - `internal/application/services/` for service interfaces/contracts (implementations live in `internal/infrastructure/services/`)
  - `internal/infrastructure/db/` for GORM DB bootstrap and runtime driver selection
  - `internal/infrastructure/persistence/models/` for GORM persistence records
  - `internal/infrastructure/persistence/repositories/` for repository implementations
  - `internal/handler/rest/` for HTTP handlers and DTOs
  - `cmd/api/` for API composition root and public route wiring
  - `cmd/backup-runner/` for internal dump/restore execution wiring

## Implementation Checklist

When adding a new DB-backed backend module, prefer this sequence:

1. `internal/domain/<entity>.go`
   Add the entity, domain errors, and any list options/results.
2. `internal/domain/<entity>_repository.go`
   Add repository interfaces used by the application layer.
3. `internal/infrastructure/persistence/models/<entity>_record.go`
   Add the GORM persistence model.
4. `internal/infrastructure/persistence/models/models.go`
   Register the new record for boot-time migration.
5. `internal/infrastructure/persistence/repositories/<entity>_repository.go`
   Implement the repository and map DB errors such as duplicate keys back to domain errors.
6. `internal/application/command/<entity>.go`
   Add write handlers.
7. `internal/application/query/<entity>.go`
   Add read handlers.
8. `internal/application/services/` plus `internal/infrastructure/services/`
   Define the service interface in `application/services/` and the concrete implementation in `infrastructure/services/`. Use this boundary only when the feature needs orchestration, provider integration, encryption, runtime control, or another non-trivial service capability.
9. `internal/handler/rest/<entity>_handler.go`
   Add request/response DTOs, route registration, error mapping, and Swagger comments.
10. `cmd/api/main.go`
   Wire repositories, services, handlers, and route registration.

## Repo Conventions

- Pagination/filter/sort for list endpoints should use `internal/contract/common.BaseRequestModel`.
- Paginated REST responses should use `internal/contract/common.BaseResponseModel`.
- Handlers should return transport DTOs, not raw domain entities.
- `FindBy...` methods are for nullable lookups; `GetBy...` methods are for required lookups that should error.
- Domain constructors and command handlers are the preferred write-path validation points.
- If both application code and the DB can detect the same duplicate condition, both should return the same domain error.
- Seed/bootstrap logic belongs in composition/bootstrap code, not in REST handlers.
- Backend logging should use `log/slog` with structured fields such as `traceId`, `method`, `path`, `status`, `err`, and stable business identifiers.
- Log system failures and unexpected infrastructure issues at `error`.
- Log expected business outcomes such as conflicts or validation failures at `info` or `warn`, or skip dedicated logs entirely if request logs already make the outcome obvious.
- Do not log the same failure in multiple layers. Prefer one boundary log near the HTTP/runtime edge unless a dedicated audit/event log is intended.
- External provider or third-party calls should log request lifecycle and failures with enough context to debug timeouts, retries, and provider-specific issues.

## Repo-Specific Notes

- The repo already follows the DDD/CQRS split for most newer modules. Extend the existing layout instead of treating it as a future target.
- Verify the actual HTTP framework in code before making framework-specific edits.
- There is version drift across some docs and build files; prefer code over prose when they conflict.
- Database backup and restore are split across `cmd/api` and `cmd/backup-runner`. Public state and orchestration stay in the API; CLI execution stays in the runner.
- Database backup support currently exists for MySQL, PostgreSQL, and MongoDB with local storage first.

## Typical Tasks

- Add or refactor domain entities and repository interfaces
- Implement command handlers and query handlers with clear separation
- Wire infrastructure repositories to domain interfaces
- Add or modify REST handlers and transport DTOs without leaking domain objects
- Add Swagger annotations and regenerate docs when REST contracts change
- Fix auth, config, persistence, or service composition issues
- Adjust Docker build, backup-runner, or service startup wiring
- Diagnose Go build, test, or runtime issues

## Validation

- Preferred checks:
  - `go test ./...`
  - `go build ./cmd/...`
  - if runner code changed, verify `go build ./cmd/backup-runner`
  - if DB/persistence changed, verify both `DB_DRIVER=sqlite` and `DB_DRIVER=postgres` startup paths when practical
- If the task affects API responses, verify handler DTOs and any frontend assumptions before concluding.
