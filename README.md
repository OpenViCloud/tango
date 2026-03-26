# tango

Monorepo: Go API + Vite React FE + Go CLI in a single Docker image.

## Core Features

- Prompt Safety
- Encrypted Secrets
- Multi-Workspace Organization
- Multi-Channel Agent Support
- Agent Orchestration
- Context Memory and Conversation Summaries
- Workflow Runs with Trace History
- Streaming Chat
- Simple Self-Hosting

## Structure

```
tango/
├── cmd/
│   ├── api/main.go        ← API server + FE serving + SPA fallback
│   └── cli/main.go        ← CLI tool
├── internal/
│   ├── auth/              ← JWT, bcrypt, middleware
│   ├── application/       ← command/query handlers
│   ├── contract/          ← shared request/response contracts
│   ├── domain/            ← entities + repository interfaces
│   ├── config/            ← shared config
│   ├── infrastructure/    ← DB bootstrap, persistence, server runtime
│   ├── messaging/         ← inbound messaging service contracts/handlers
│   ├── channels/          ← transport adapters (Discord, ...)
│   └── handler/           ← HTTP handlers
├── web/                   ← Vite + React + TanStack
├── Dockerfile             ← multi-stage, output ~15MB
├── docker-compose.yml     ← app + postgres + redis
├── Makefile
└── install.sh             ← install the CLI for end users
```

## Backend Conventions

The backend in this repo follows a pragmatic DDD/CQRS split around the current layout:

- `internal/domain/`
  Entities, domain errors, list options/results, repository interfaces
- `internal/application/command/`
  Write use cases such as create, update, delete, ban, assign
- `internal/application/query/`
  Read use cases such as get by id and list
- `internal/application/services/`
  Application-facing service contracts when the feature is not a simple command/query flow
- `internal/infrastructure/persistence/models/`
  GORM persistence records used by `AutoMigrate()`
- `internal/infrastructure/persistence/repositories/`
  Repository implementations and DB error mapping
- `internal/infrastructure/services/`
  Service implementations that talk to providers, encryption, runtime systems, or other infrastructure
- `internal/handler/rest/`
  HTTP handlers, request DTOs, response DTOs, error mapping, and Swagger annotations
- `cmd/api/main.go`
  Composition root: instantiate repositories/services/handlers and register routes

### Adding a new backend module

When adding a new DB-backed module such as `role`, `channel`, or `foo`, follow this order:

1. Add the domain entity and domain errors in `internal/domain/`
2. Add the repository interface in `internal/domain/`
3. Add the GORM record in `internal/infrastructure/persistence/models/`
4. Register the new record in `internal/infrastructure/persistence/models/models.go`
5. Implement the repository in `internal/infrastructure/persistence/repositories/`
6. Add write use cases in `internal/application/command/` and read use cases in `internal/application/query/`
7. If the feature needs orchestration or provider-specific logic, add an application service contract in `internal/application/services/` and the implementation in `internal/infrastructure/services/`
8. Add the REST handler in `internal/handler/rest/`
9. Wire everything in `cmd/api/main.go`
10. Add Swagger annotations and regenerate `docs/`
11. Run `go test ./...`

### Rules of thumb

- Put validation and invariants on the write path in domain constructors or command handlers
- Do not return raw domain entities from REST handlers; map them to transport DTOs
- Reuse `internal/contract/common.BaseRequestModel` and `BaseResponseModel` for paginated list endpoints
- Use one canonical domain error per business case, then map it to one public API error code
- Keep DB unique constraints as the final line of defense and map duplicate-key errors back to the same domain error used by application pre-checks
- Use `FindBy...` for nullable lookups and `GetBy...` for required lookups
- Add Swagger comments for every public REST endpoint and regenerate docs after changing the API

## Context Memory

Tango persists lightweight execution memory so orchestrators and child agents can reuse stable context across turns without replaying an entire raw conversation every time.

### Context scopes currently used at runtime

- `workspace`
  Shared memory for the workspace entry-agent path
- `conversation`
  Conversation-level summary used only when the thread is deep enough to justify summarization
- `agent_conversation`
  Memory specific to one child agent inside one conversation

Notes:

- `task`, `agent_task`, and `agent` exist in the domain model but are not wired into the active orchestration path yet.
- The current deep-thread threshold is `10` messages. It is implemented as a constant today and can be promoted to config later.
- `agent_conversation` summaries include both the delegated task input and the agent output so sub-agents can reuse prior work across turns.
- `runAgent()` calls `loadAgentCache()` before invoking the LLM. The cache is a no-op placeholder today; swap the implementation to DB, Redis, or in-memory cache later without changing callers.

### Context tables

#### `contexts`

Stores the latest active memory snapshot for one scope.

| Column                                     | Description                                                                                  |
| ------------------------------------------ | -------------------------------------------------------------------------------------------- |
| `id`                                       | Context record ID                                                                            |
| `workspace_id`                             | Owning workspace                                                                             |
| `conversation_id`                          | Optional conversation scope key                                                              |
| `task_id`                                  | Optional reserved task scope key                                                             |
| `agent_id`                                 | Optional agent scope key                                                                     |
| `scope_type`                               | Scope discriminator such as `workspace`, `conversation`, or `agent_conversation`             |
| `status`                                   | Current context status                                                                       |
| `summary`                                  | Prompt-ready summary text                                                                    |
| `state_json`                               | Structured state payload such as durable facts, open items, decisions, and current task info |
| `version`                                  | Monotonic version, incremented only when summary or state changes                            |
| `created_at` / `updated_at` / `deleted_at` | Lifecycle fields                                                                             |

#### `context_histories`

Stores immutable snapshots of the previous context state before an update changes the current record.

| Column         | Description                   |
| -------------- | ----------------------------- |
| `id`           | History record ID             |
| `context_id`   | Parent context record         |
| `workspace_id` | Owning workspace              |
| `scope_type`   | Scope at the time of snapshot |
| `summary`      | Previous summary              |
| `state_json`   | Previous structured state     |
| `version`      | Version that was replaced     |
| `created_at`   | Snapshot timestamp            |

## Message And Memory Flow

### Orchestrator Message Flow

This diagram shows how an incoming user message resolves the entry agent, conditionally loads workspace and conversation summaries, delegates to child agents when needed, and persists memory after the run completes.

```mermaid
flowchart TD
    A["Incoming message / user request"] --> B["Build conversation history"]
    B --> C["Resolve runtime"]
    C --> D["Select entry agent"]
    D --> E["loadOrchestratorPromptContext()"]

    E --> E1["Load workspace summary"]
    E --> E2{"message count > 10? (configurable threshold)"}
    E2 -- "No" --> F["Build entry-agent LLM"]
    E2 -- "Yes" --> E3["Load conversation summary"]
    E1 --> F
    E3 --> F

    F --> G{"Single active agent?"}
    G -- "Yes" --> H["Generate direct response"]
    G -- "No" --> I{"Root orchestrator with child agents?"}

    I -- "No" --> J["Fallback direct response"]
    I -- "Yes" --> K["Delegation loop"]

    K --> L["Orchestrator decides next action"]
    L --> M{"call_agent or final?"}

    M -- "final" --> N["Return final response"]

    M -- "call_agent" --> O["runAgent(child)"]
    O --> O1{"loadAgentCache() hit?"}
    O1 -- "Yes" --> O2["Return cached output"]
    O2 --> K
    O1 -- "No" --> P["loadSubAgentPromptContext()"]
    P --> Q["Load agent_conversation summary"]
    Q --> R["Build child-agent LLM with memoryContext"]
    R --> S["Child generates output"]
    S --> T["persistAgentContext()"]
    T --> T1["Upsert agent_conversation summary (input + output)"]
    T1 --> K

    H --> U["finishTrace()"]
    J --> U
    N --> U

    U --> V["persistExecutionContexts()"]
    V --> W["Upsert workspace summary"]
    V --> X["Upsert conversation summary"]
```

### Orchestrator Prompt Context Load

This diagram shows the orchestrator-specific prompt context path. The entry agent always loads `workspace` memory and only loads `conversation` memory when the message count is above the deep-thread threshold.

```mermaid
flowchart TD
    A["Orchestrator path"] --> B["BuildPromptContext(input)"]
    B --> C["workspace lookup"]
    B --> D{"message count > 10? (configurable threshold)"}
    D -- "No" --> E["Skip conversation lookup"]
    D -- "Yes" --> F["conversation lookup"]
    C --> G["contexts.GetByScope(...)"]
    F --> G
    G --> H{"Summary exists?"}
    H -- "No" --> I["Skip"]
    H -- "Yes" --> J["Append summary to prompt"]
    J --> K["Return Relevant execution context"]
```

### Sub-Agent Prompt Context Load

This diagram shows the child-agent-specific prompt context path. A sub-agent does not load shared workspace or conversation memory. It only loads its own `agent_conversation` memory. The summary includes both the delegated task input and the agent output so the agent can reuse prior work across turns.

```mermaid
flowchart TD
    A["Sub-agent path"] --> B["BuildPromptContext(input)"]
    B --> C["agent_conversation lookup"]
    C --> D["contexts.GetByScope(...)"]
    D --> E{"Summary exists?"}
    E -- "No" --> F["Skip"]
    E -- "Yes" --> G["Append summary to prompt"]
    G --> H["Return Relevant execution context"]
```

### Context Persistence And Summary Update

This diagram shows how Tango builds or updates one context record, when conversation summarization kicks in, and when a history snapshot is created before the active summary changes.

```mermaid
flowchart TD
    A["UpsertExecutionContext(run, steps, scope)"] --> B["Build executionSummary"]
    B --> C["Load existing context by scope"]
    C --> D["Parse existing state_json"]
    D --> E["mergeContextState(...)"]

    E --> F["buildSummaryForScope(...)"]
    F --> G{"scope == conversation?"}

    G -- "No" --> H["buildContextSummary(...)"]

    G -- "Yes" --> I["Count conversation messages"]
    I --> J{"message count > 10? (configurable threshold)"}
    J -- "No" --> H
    J -- "Yes" --> K["Load archived messages before recent raw tail"]
    K --> L["summarizeConversationMessages(...)"]
    L --> H

    H --> M["Marshal next state JSON"]
    M --> N{"Existing context?"}

    N -- "No" --> O["Create new context version = 1"]

    N -- "Yes" --> P{"Summary/state changed?"}
    P -- "No" --> Q["Return existing, no version bump"]
    P -- "Yes" --> R["snapshotContext(old version)"]
    R --> S["Update context version + 1"]
```

## Requirements

- Go 1.22+
- Node.js 20+ and pnpm
- Docker + Docker Compose

## Run Locally

### 1. Install Go

```bash
brew install go
go version  # go version go1.22.x darwin/arm64
```

### 2. Install dependencies

```bash
# Go dependencies
cd tango
go mod tidy

# FE dependencies
cd web && pnpm install && cd ..
```

### 3. Set environment variables

```bash
export JWT_SECRET=mysecretkey123
export PORT=8080
export DB_DRIVER=postgres
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/tango?sslmode=disable
export LLM_CONFIG_ENCRYPTION_KEY=12345678901234567890123456789012
export ORCHESTRATION_ENGINE=custom
export WORKFLOW_ENGINE=custom
```

Local SQLite:

```bash
export DB_DRIVER=sqlite
export DATABASE_URL='file:tango.db?_foreign_keys=on'
```

Optional Discord runtime:

```bash
export DISCORD_BOT_TOKEN='your_discord_bot_token'
export DISCORD_REQUIRE_MENTION=true
export DISCORD_ENABLE_TYPING=true
export DISCORD_ENABLE_MESSAGE_CONTENT_INTENT=false
export DISCORD_ALLOWED_USER_IDS='123456789012345678,987654321098765432'
```

Optional Telegram runtime:

```bash
export TELEGRAM_BOT_TOKEN='your_telegram_bot_token'
export TELEGRAM_ENABLE_TYPING=true
export TELEGRAM_ALLOWED_USER_IDS='123456789,987654321'
```

Notes:

- `LLM_CONFIG_ENCRYPTION_KEY` is required if you want to store LLM API keys in the DB.
- The key must be exactly `32` characters long.
- The app no longer reads `DEEPSEEK_API_KEY` directly for Discord/Telegram replies.
- The active LLM provider is now managed through the API, and its API key is encrypted before being stored in the DB.
- `ORCHESTRATION_ENGINE` controls chat/workspace multi-agent orchestration:
  - `custom`: Tango's native orchestration loop for `workspace -> orchestrator -> child agents`
  - `eino`: Eino-backed orchestration graph for chat delegation
- `WORKFLOW_ENGINE` controls workflow DAG execution via `/api/workflows/:id/execute`:
  - `custom`: Tango's native DAG executor
  - `eino`: Eino DAG graph execution using workflow nodes and edges from the DB
- If not set, both `ORCHESTRATION_ENGINE` and `WORKFLOW_ENGINE` default to `custom`.

Examples:

```bash
# Default/native engines for both chat orchestration and workflow DAG execution
export ORCHESTRATION_ENGINE=custom
export WORKFLOW_ENGINE=custom

# Eino for both chat orchestration and workflow DAG execution
export ORCHESTRATION_ENGINE=eino
export WORKFLOW_ENGINE=eino

# Keep chat orchestration on Tango native loop, but run workflow DAGs on Eino
export ORCHESTRATION_ENGINE=custom
export WORKFLOW_ENGINE=eino
```

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
# Build the image and start all services
docker compose up --build

# Access
# http://localhost:8080 → Web UI + API
# demo login:
# email: demo.admin@example.com
# password: password123
```

## Build And Test The CLI On macOS

### Build the binary

```bash
cd tango
go build -o bin/demo ./cmd/cli
```

### Install into PATH

```bash
sudo cp bin/demo /usr/local/bin/demo
# or without sudo
mkdir -p ~/bin && cp bin/demo ~/bin/demo
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

### Test the CLI

```bash
# Check the version
demo version

# Check whether the server is running
demo status

# Log in (stores the token in ~/.config/demo/credentials.json)
demo login

# Show the current user (requires login first)
demo user me

# Get a user by ID
demo user get 123

# Log out
demo logout
```

### Manage Discord runtime from the CLI

```bash
# Show Discord runtime state on the API server
demo discord status

# Start or replace the runtime config
demo discord start --token 'your_discord_bot_token'

# Restart with the config currently kept in API memory
demo discord restart

# Stop the runtime
demo discord stop
```

### Build the CLI for all OS targets

```bash
make build-cli
# outputs to ./bin/
# demo_linux_amd64
# demo_linux_arm64
# demo_darwin_amd64
# demo_darwin_arm64
# demo_windows_amd64.exe
```

## Install The CLI For End Users (Production)

```bash
curl -fsSL https://yourdomain.com/install.sh | bash
demo status
```

## API Endpoints

| Method | Path                         | Auth | Description                          |
| ------ | ---------------------------- | ---- | ------------------------------------ |
| GET    | /api/status                  | ❌   | Server status                        |
| POST   | /api/auth/login              | ❌   | Log in                               |
| POST   | /api/auth/refresh            | ❌   | Refresh token                        |
| POST   | /api/auth/logout             | ❌   | Log out                              |
| GET    | /api/user/me                 | ✅   | Current user                         |
| GET    | /api/user/:id                | ✅   | User by ID                           |
| GET    | /api/runtime/discord/status  | ✅   | Discord runtime status               |
| POST   | /api/runtime/discord/start   | ✅   | Start/reload Discord runtime         |
| POST   | /api/runtime/discord/restart | ✅   | Restart Discord runtime              |
| POST   | /api/runtime/discord/stop    | ✅   | Stop Discord runtime                 |
| GET    | /api/llm/providers           | ✅   | List LLM providers                   |
| GET    | /api/llm/providers/:id       | ✅   | LLM provider by ID                   |
| POST   | /api/llm/providers           | ✅   | Create an LLM provider               |
| PUT    | /api/llm/providers/:id       | ✅   | Update an LLM provider               |
| DELETE | /api/llm/providers/:id       | ✅   | Delete an LLM provider               |
| GET    | /api/roles                   | ✅   | List roles                           |
| GET    | /api/roles/:id               | ✅   | Role by ID                           |
| POST   | /api/roles                   | ✅   | Create role                          |
| PUT    | /api/roles/:id               | ✅   | Update role                          |
| DELETE | /api/roles/:id               | ✅   | Delete role                          |
| POST   | /api/chat/completions        | ✅   | Chat completion, optional SSE stream |
| GET    | /api/chat/stream             | ✅   | Demo SSE stream                      |

## Database

- The app uses `GORM`.
- Select the runtime DB with `DB_DRIVER=postgres|sqlite`.
- The app creates the PostgreSQL database automatically if it does not exist.
- The app runs `AutoMigrate()` on boot based on the persistence records in:
  - `internal/infrastructure/persistence/models`
- Tables currently managed by GORM:
  - `users`
  - `roles`
  - `user_roles`
  - `llm_providers`
  - `provider_credentials`
  - `workspaces`
  - `workspace_members`
  - `channels`
  - `agents`
  - `agent_providers`
  - `skills`
  - `agent_skills`
  - `conversations`
  - `conversation_messages`
  - `contexts`
  - `context_histories`
  - `workflows`
  - `workflow_nodes`
  - `workflow_edges`
  - `runs`
  - `run_steps`
  - `pipelines`
  - `pipeline_stages`
  - `tasks`
  - `task_stage_histories`

Notes:

- The old `migrations/` directory is no longer used at runtime.
- Simple schema changes such as adding a new field can rely on `AutoMigrate()`.
- For breaking schema changes or complex data migrations, you should still write a dedicated migration/script.
- When adding a new schema, add the corresponding GORM record to `internal/infrastructure/persistence/models/models.go` so boot-time migration picks it up.

## LLM Configuration

The app supports managing multiple `LLM provider` records in the DB. API keys are encrypted with `LLM_CONFIG_ENCRYPTION_KEY` before being stored.

Example: create a DeepSeek provider:

```bash
curl -X POST http://localhost:8080/api/llm/providers \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "deepseek-primary",
    "provider": "deepseek",
    "model": "deepseek-chat",
    "api_key": "your_deepseek_api_key",
    "base_url": "https://api.deepseek.com",
    "active": true,
    "primary": true
  }'
```

List providers:

```bash
curl 'http://localhost:8080/api/llm/providers?pageIndex=0&pageSize=20&orderBy=name&ascending=true' \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

After a successful save:

- The API rebuilds `LLMService` from the provider where `active=true`.
- Discord and Telegram inbound replies start using the new active provider immediately.
- No app restart is required just to change the model or API key.

# NEW FOR CLOUD

docker run -d \
 --name buildkitd \
 --privileged \
 -p 1234:1234 \
 moby/buildkit:latest \
 --addr tcp://0.0.0.0:1234

# check

docker exec buildkitd buildctl debug workers

# setup and run

export BUILDKIT_HOST=tcp://localhost:1234
export BUILD_WORKSPACE_DIR=/tmp/tango-builds
go run ./cmd/api
