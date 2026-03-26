# Tango Go Backend — Architecture & Implementation Plan

> This document captures the target system design, module breakdown, and implementation roadmap for the Tango Go backend, with a focus on the agent orchestration layer.

> Note: this file describes the target architecture. The pragmatic phase-one cut is documented separately in `roadmap/MVP_SCOPE.md`.

---

## Scope Note

For the practical MVP scope, see `roadmap/MVP_SCOPE.md`.

This architecture document should be read as the longer-term direction, not as a statement that every subsystem below is already implemented in the current product.

The current Phase 1 foundation is workspace-centric and uses `llm_providers` plus `agent_providers` for agent-level primary and fallback routing.

For multi-agent execution, Tango should keep two separate structures:

- `agents.parent_agent_id` and `agents.kind` for supervisor tree orchestration
- `workflows`, `workflow_nodes`, and `workflow_edges` for DAG-style sequential and parallel execution

Execution history should be persisted through `runs` and `run_steps`, not mixed into conversation messages.

---

## System Overview

Tango is a platform for orchestrating AI agents to help run an autonomous company. The Go backend is responsible for **agent orchestration**. It does not replace the UI or the external agent runtimes.

```mermaid
graph TD
    UI[React UI] --> API[Go Backend API]
    API --> CM[Company Module]
    API --> AM[Agent Registry]
    API --> SM[Skill System]
    API --> CE[Context Engine]
    CE --> AR[Agent Runtime]
    AR -->|self-evolution feedback| CE
```

---

## Multi-Agent Execution Model

Tango should support both supervisor-style delegation and DAG-style workflow execution.

### Supervisor Tree

- `workspace` contains many `agents`
- `agents.kind` distinguishes `orchestrator` from `worker`
- `agents.parent_agent_id` forms a direct-child hierarchy
- the runtime appends child-agent context into the active agent system prompt
- this model is appropriate for dynamic delegation such as:
  - `CEO -> BA -> Tech Lead -> FE Dev`
  - `CEO -> Writer -> Reviewer`

### Workflow DAG

- fixed execution graphs should be modeled explicitly, not inferred from `parent_agent_id`
- `workflow_nodes` reference `agents`
- `workflow_edges` describe dependencies and execution relationships
- this model is appropriate for deterministic flows such as:
  - `Research -> Image -> Post -> Publish`
  - `Research -> (Image || Post) -> Publish`

### Why Both Exist

- hierarchy answers who may delegate to whom
- workflow graph answers what depends on what
- a single `parent_agent_id` field is not enough to model joins, fan-out, fan-in, or explicit parallel branches

### Channel Routing

A channel declares its orchestration target via one of two nullable fields:

| Field | Target type | Runtime |
|---|---|---|
| `channels.workspace_id` | `TeamTarget` | Multi-agent workspace orchestration |
| `channels.target_agent_id` | `AgentTarget` | Direct single-agent, no orchestration layer |

Exactly one should be set; `target_agent_id` takes priority if both are present.

When a conversation is created from a channel, it copies the target from the channel at that point in time (`conversation.workspace_id` / `conversation.target_agent_id`). This means the conversation is self-contained — it does not re-read the channel on every message.

`conversationOrchestrationTarget()` inspects the conversation and returns either an `AgentTarget` or a `TeamTarget`, which the `OrchestrationService` then dispatches to `resolveAgentTarget` or `resolveTeamTarget` accordingly.

### Execution History

- `conversations` and `conversation_messages` remain user-facing history and LLM context
- `runs` and `run_steps` capture orchestration and workflow execution traces
- `run_steps` should store decision, call, result, final, and error events for debugging and future Kanban views

### Planned Runtime Services

- `OrchestrationService`
  - entry agent resolution
  - supervisor loop execution
  - dynamic child delegation
- `WorkflowService`
  - workflow, node, and edge CRUD
  - graph validation
  - graph data for UI rendering
- `WorkflowExecutionService`
  - load workflow definitions from DB
  - run sequential and parallel execution
  - future Eino integration point
- `RunTraceService`
  - persist `runs` and `run_steps`
  - expose execution traces to debug and operational UI

### Target Schema Additions

The ERD below mirrors the canonical schema diagram in [erd.mmd](/Users/felix/projects/tango/web/docs/erd.mmd). Update `web/docs/erd.mmd` first when the schema changes, then sync this section.

```mermaid
erDiagram

  %% IDENTITY
  users {
    varchar id PK
    text email
    text nickname
    text first_name
    text last_name
    text phone
    text address
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

  %% WORKSPACE
  workspaces {
    varchar id PK
    varchar name
    varchar description
    varchar status
    text metadata_json
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  workspace_members {
    varchar id PK
    varchar workspace_id FK
    text user_id FK
    varchar member_role
    timestamptz created_at
  }

  %% LLM PROVIDERS
  llm_providers {
    text id PK
    text name
    text provider
    text model
    text encrypted_api_key
    text base_url
    bool is_active
    bool is_primary
    timestamptz created_at
    timestamptz updated_at
  }
  provider_credentials {
    varchar id PK
    varchar provider_id FK
    varchar kind
    text encrypted_api_key
    text encrypted_access_token
    text encrypted_refresh_token
    varchar token_type
    text scope
    timestamptz expires_at
    timestamptz refresh_expires_at
    text encrypted_metadata_json
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }

  %% AGENTS
  agents {
    varchar id PK
    varchar workspace_id FK
    varchar parent_agent_id FK
    varchar name
    varchar role
    varchar kind
    varchar type
    varchar status
    text system_prompt
    varchar model_override
    float temperature
    int max_tokens
    bigint budget_limit
    bigint budget_used
    text metadata_json
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  agent_providers {
    varchar id PK
    varchar agent_id FK
    varchar llm_provider_id FK
    int priority
    varchar status
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }

  %% CHANNELS & CONVERSATIONS
  channels {
    text id PK
    varchar workspace_id FK "team target (nullable)"
    varchar target_agent_id FK "direct agent target (nullable)"
    text name
    text kind
    text status
    text encrypted_credentials
    text settings_json
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  conversations {
    text id PK
    text workspace_id FK "team target (nullable)"
    text target_agent_id FK "direct agent target (nullable)"
    text channel_id FK
    text user_id FK
    text channel_kind
    text conversation_type
    text external_chat_id
    text title
    text status
    bool auto_reply_enabled
    timestamptz last_message_at
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  conversation_messages {
    text id PK
    text conversation_id FK
    text run_id FK
    text external_message_id
    int sequence
    text sender_type
    text sender_id
    varchar message_kind
    text role
    text content
    text status
    text provider
    text model
    text finish_reason
    text error_message
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }

  %% WORKFLOWS
  workflows {
    varchar id PK
    varchar workspace_id FK
    varchar name
    text description
    varchar status
    varchar trigger_mode
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  workflow_nodes {
    varchar id PK
    varchar workflow_id FK
    varchar agent_id FK
    varchar name
    varchar node_type
    float position_x
    float position_y
    text config_json
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  workflow_edges {
    varchar id PK
    varchar workflow_id FK
    varchar from_node_id FK
    varchar to_node_id FK
    varchar execution_mode
    varchar label
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }

  %% RUNS
  runs {
    varchar id PK
    varchar workspace_id FK
    varchar workflow_id FK
    varchar conversation_id FK
    varchar entry_agent_id FK
    varchar status
    text input
    text final_output
    timestamptz started_at
    timestamptz finished_at
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  run_steps {
    varchar id PK
    varchar run_id FK
    varchar agent_id FK
    varchar workflow_node_id FK
    int step_index
    varchar step_type
    varchar status
    text input
    text output
    text metadata_json
    timestamptz started_at
    timestamptz finished_at
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }

  %% PIPELINES
  pipelines {
    varchar id PK
    varchar workspace_id FK
    varchar name
    varchar type
    varchar status
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  pipeline_stages {
    varchar id PK
    varchar pipeline_id FK
    varchar name
    int position
    varchar stage_type
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }

  %% TASKS
  tasks {
    varchar id PK
    varchar workspace_id FK
    text conversation_id FK
    varchar pipeline_id FK
    varchar agent_id FK
    varchar parent_task_id FK
    varchar current_stage_id FK
    text created_by
    varchar title
    text description
    varchar status
    varchar priority
    timestamptz created_at
    timestamptz updated_at
    timestamptz deleted_at
  }
  task_stage_history {
    varchar id PK
    varchar task_id FK
    varchar from_stage_id FK
    varchar to_stage_id FK
    varchar changed_by_type
    varchar changed_by_id
    text note
    timestamptz created_at
  }

  %% RELATIONSHIPS
  users ||--o{ user_roles : ""
  roles ||--o{ user_roles : ""
  users ||--o{ workspace_members : ""
  workspaces ||--o{ workspace_members : ""

  llm_providers ||--o{ provider_credentials : ""
  llm_providers ||--o{ agent_providers : ""
  agents ||--o{ agent_providers : ""

  workspaces ||--o{ agents : ""
  agents ||--o{ agents : "parent"

  workspaces ||--o{ channels : ""
  channels ||--o{ conversations : ""
  users ||--o{ conversations : ""
  conversations ||--o{ conversation_messages : ""

  workspaces ||--o{ workflows : ""
  workflows ||--o{ workflow_nodes : ""
  workflows ||--o{ workflow_edges : ""
  workflow_nodes ||--o{ workflow_edges : "from"
  workflow_nodes ||--o{ workflow_edges : "to"
  agents ||--o{ workflow_nodes : ""

  workspaces ||--o{ runs : ""
  workflows ||--o{ runs : ""
  conversations ||--o{ runs : ""
  agents ||--o{ runs : "entry"
  runs ||--o{ run_steps : ""
  agents ||--o{ run_steps : ""
  workflow_nodes ||--o{ run_steps : ""
  runs ||--o{ conversation_messages : ""

  workspaces ||--o{ pipelines : ""
  pipelines ||--o{ pipeline_stages : ""

  workspaces ||--o{ tasks : ""
  conversations ||--o{ tasks : ""
  pipelines ||--o{ tasks : ""
  agents ||--o{ tasks : ""
  tasks ||--o{ tasks : "parent"
  pipeline_stages ||--o{ tasks : "current stage"
  users ||--o{ tasks : "creates"
  tasks ||--o{ task_stage_history : ""
  pipeline_stages ||--o{ task_stage_history : "from"
  pipeline_stages ||--o{ task_stage_history : "to"
```

---

## Module Architecture

### Five Main Layers

```mermaid
graph TD
    C[Company\nMission · Budget · Multi-tenant isolation]
    C --> OC[Org Chart\nHierarchy · Roles · Reporting]
    C --> GT[Goal Tree\nCompany → Project → Task]
    OC --> AR[Agent Registry\nType · Role · Parent · Budget · Status]
    GT --> AR
    AR --> HB[Heartbeat\nSchedule · Wake · Sleep]
    AR --> EX[Executor\nCheckout · Run · Result]
    AR --> BG[Budget Enforcer\nAtomic token tracking]
    HB --> CE[Context Engine\nGoal tree · Skills · Memory · Org · State]
    EX --> CE
    BG --> CE
    CE --> P[Agent Protocol\nHeartbeat contract · Response schema]
    CE --> AL[Audit Log\nImmutable · Tool-call trace]
    P --> RT[Agent Runtime\nLLM · Bash · HTTP]
```

---

## Module Details

### 1. Company

This is the root of the system. Every entity belongs to a single company.

| Field | Description |
|---|---|
| `id` | UUID |
| `name` | Company name |
| `mission` | Top-level mission |
| `budget_total` | Total budget |
| `budget_used` | Consumed budget |
| `owner_id` | Owning user |

**Important:** Cross-company isolation must be enforced at the database query level. Every relevant query should filter by `company_id`.

---

### 2. Org Chart + Goal Tree

```mermaid
graph TD
    CEO[CEO Agent] --> CTO[CTO Agent]
    CEO --> CMO[CMO Agent]
    CTO --> E1[Engineer 1]
    CTO --> E2[Engineer 2]
    CMO --> MK[Marketer]
```

```mermaid
graph TD
    M[Company Mission\n'Build the #1 AI note-taking app'] --> P1[Project: MVP]
    M --> P2[Project: Marketing]
    P1 --> T1[Task: Auth system]
    P1 --> T2[Task: Note editor]
    T1 --> S1[Subtask: Login page]
    T1 --> S2[Subtask: JWT implementation]
```

The **Goal Tree** ensures that an agent always knows the reason behind the work. Each task carries its full ancestry from mission to execution.

---

### 3. Agent Registry

```mermaid
graph LR
    AR[Agent Registry] --> AT[Agent Types\nLLM · Bash · HTTP]
    AR --> AS[Agent Status\nactive · paused · terminated]
    AR --> AB[Agent Budget\nlimit · used · reset_at]
    AR --> ASK[Agent Skills\nAssigned skill versions]
```

| Field | Description |
|---|---|
| `id` | UUID |
| `company_id` | Owning company |
| `name` | Agent name |
| `role` | CEO / CTO / Engineer... |
| `type` | llm / bash / http |
| `parent_id` | Reporting manager agent |
| `budget_limit` | Monthly token budget |
| `budget_used` | Consumed budget |
| `status` | active / paused / terminated |
| `skills` | Assigned skill version IDs |

---

### 4. Heartbeat + Executor + Budget Enforcer

```mermaid
sequenceDiagram
    Scheduler->>HeartbeatEngine: Wake agent (cron)
    HeartbeatEngine->>Executor: CheckoutTask(agentID)
    Executor->>DB: SELECT FOR UPDATE SKIP LOCKED
    DB-->>Executor: Task (atomic)
    Executor->>BudgetEnforcer: CheckBudget(agentID)
    BudgetEnforcer-->>Executor: OK / EXCEEDED
    Executor->>ContextEngine: Build(agentID, taskID)
    ContextEngine-->>Executor: ContextPackage
    Executor->>AgentRuntime: Send(payload)
    AgentRuntime-->>Executor: Result
    Executor->>BudgetEnforcer: DeductTokens(used)
    Executor->>AuditLog: Record(event)
```

**Atomic checkout:** use `SELECT ... FOR UPDATE SKIP LOCKED` so two agents cannot claim the same task.

---

### 5. Skill System

#### Three Pillars

```mermaid
graph TD
    SS[Skill System]
    SS --> SE[Self-Evolution\nAgent proposes changes\nUser approves/rejects]
    SS --> SL[Skill Learning & Management\nCreate · Assign · Search · Version]
    SS --> SV[Security & Versioning\nAccess control · Content scan\nRollback · Audit]
```

#### Skill Struct

| Field | Description |
|---|---|
| `id` | UUID |
| `owner_id` | Creating user |
| `company_id` | Null for private/public scope |
| `name` | Skill name |
| `content` | SKILL.md content |
| `tags` | Used to match tasks |
| `compatible_agent_types` | llm / bash / http... |
| `visibility` | private / company / public |
| `version` | Current version number |
| `content_hash` | SHA256 for tamper resistance |
| `status` | draft / pending_review / approved / revoked |

#### Visibility Flow

```mermaid
stateDiagram-v2
    [*] --> draft: User creates skill
    draft --> pending_review: Submit public skill
    draft --> approved: Private/company auto-approval
    pending_review --> approved: Admin approval
    pending_review --> rejected: Admin rejection
    approved --> revoked: Malicious content detected
    revoked --> [*]
```

#### Self-Evolution Flow

```mermaid
sequenceDiagram
    Agent->>EvolutionService: ProposeChange(taskResult, currentSkills)
    EvolutionService-->>User: SkillProposal (diff + reason)
    User->>EvolutionService: Approve / Reject
    EvolutionService->>SkillService: ApplyProposal (if approved)
    SkillService->>AuditLog: Record(actor=agentID)
```

An agent **cannot apply its own changes**. Every change must pass through an approval gate.

#### Security Layers

```mermaid
graph TD
    A[User submits skill] --> B[SkillValidator.Scan\nDetect prompt injection]
    B --> C{Visibility?}
    C -->|public| D[pending_review\nAdmin approval]
    C -->|private/company| E[Auto approved]
    D --> F[Approved + content_hash]
    E --> F
    F --> G[AuditLog.Record]

    H[Malicious skill detected] --> I[RevokeVersion]
    I --> J[Force rollback for all agents using this version]
    J --> K[Notify owner + AuditLog]
```

---

### 6. Context Engine

**Context engineering** does more than send a task. It builds the full working environment for the agent before execution.

```mermaid
graph TD
    IN[agentID + taskID] --> CE[Context Engine]

    CE --> GT[Goal Tree Traverser\nTask → Project → Mission]
    CE --> SR[Skill Resolver\nMatch skills by task type]
    CE --> MS[Memory Search\nVector similarity - pgvector]
    CE --> OR[Org Resolver\nRole · Manager · Reports]
    CE --> ST[State Reader\nBudget · Sprint · Flags]

    GT --> CP[Context Packer\nRank + Trim → fit token limit]
    SR --> CP
    MS --> CP
    OR --> CP
    ST --> CP

    CP --> PKG[Context Package]
    PKG --> RT[Agent Runtime]
    RT -->|feedback| EV[Self-Evolution signal]
```

#### Context Package Structure

```text
[System layer]       Role · Skills · Behavior rules (from evolution)
[Company layer]      Mission → Project → Task (goal tree)
[Memory layer]       Top-N relevant past tickets (vector search)
[State layer]        Budget · Org position · Sprint
[Task layer]         Task detail · Dependencies · Related tasks
```

#### Context Packer Priority Rules

| Priority | Layer | Action |
|---|---|---|
| 1 — never trim | Task + Goal tree | Keep intact |
| 2 — trim if needed | Skills | Keep top 3 most relevant |
| 3 — trim more aggressively | Memory | Keep top 5 |
| 4 — summarize | Org + Budget | Compress to 1-2 lines |

---

## Go Package Structure

```text
internal/
├── company/
│   ├── domain.go
│   ├── repository.go
│   └── service.go
├── org/
│   ├── chart.go          ← hierarchy, reporting lines
│   └── goal_tree.go      ← ancestry traversal
├── agent/
│   ├── registry.go
│   ├── heartbeat.go      ← cron scheduler, goroutine per agent
│   ├── executor.go       ← atomic checkout, run, result
│   └── budget.go         ← atomic token tracking
├── skill/
│   ├── domain.go
│   ├── service.go        ← interface for legacy integrations
│   ├── evolution.go      ← proposal, diff, approve/reject
│   ├── versioning.go     ← snapshot, rollback
│   ├── security.go       ← access control, content scan
│   ├── validator.go      ← prompt injection detection
│   └── repository.go
├── context/
│   ├── engine.go         ← orchestrates the pipeline
│   ├── goal_tree.go      ← traverses ancestry
│   ├── skill_resolver.go ← matches skills to tasks
│   ├── memory.go         ← vector search (pgvector)
│   ├── org_resolver.go
│   └── packer.go         ← rank + trim → token limit
├── protocol/
│   ├── heartbeat.go      ← payload contract
│   └── response.go       ← response schema
└── audit/
    └── log.go            ← immutable event log
```

---

## Tech Stack

| Concern | Library |
|---|---|
| HTTP router | `gin-gonic/gin` |
| PostgreSQL driver | `pgx/v5` |
| Type-safe SQL | `sqlc` |
| Vector embedding | `pgvector/pgvector-go` + pgvector extension |
| Scheduler | `robfig/cron/v3` |
| Goroutine management | `golang.org/x/sync/errgroup` |
| WebSocket | `gorilla/websocket` |
| Config | `spf13/viper` |
| Logging | `go.uber.org/zap` |
| Tracing | `go.opentelemetry.io/otel` |
| Testing | `testify` |

---

## Eino Integration Strategy

Tango should treat `cloudwego/eino` as an **agent runtime framework**, not as the full orchestration backend.

### Recommended Boundary

**Tango owns the control plane:**

- Company and tenant isolation
- Agent registry
- Goal tree and task lifecycle
- Run records and retries
- Budget enforcement
- Skill assignment and approval workflows
- Audit log and policy enforcement

**Eino owns the execution plane:**

- Agent and sub-agent runtime execution
- Tool calling loops
- Supervisor, plan-execute, and deep-agent coordination
- Interrupt/resume for human-in-the-loop
- Checkpoint-aware execution state
- Runtime callbacks and streaming events

### Why This Split Works

Eino provides strong Go-native primitives for building and running agents, including multi-agent collaboration, callbacks, interrupts, resume flows, and checkpoint support. These capabilities are useful for the runtime layer inside Tango.

However, Tango still needs its own persistent orchestration model. Multi-tenant isolation, budget policy, task claiming, run history, approval gates, and auditability are product-specific concerns and should remain inside Tango's domain and application layers.

### Integration Flow

```mermaid
sequenceDiagram
    TangoAPI->>Orchestrator: StartRun(agentID, taskID)
    Orchestrator->>ContextEngine: BuildContext(agentID, taskID)
    ContextEngine-->>Orchestrator: ContextPackage
    Orchestrator->>EinoRuntime: Execute(context, tools, policy)
    EinoRuntime-->>Orchestrator: Stream events / interrupt / result
    Orchestrator->>BudgetService: Deduct usage
    Orchestrator->>AuditLog: Record run events
    Orchestrator-->>TangoAPI: Final status
```

### Practical Guidance

- Use Eino runners, agent patterns, and callbacks behind an internal Tango runtime adapter.
- Do not let Eino define Tango's database schema or domain lifecycle.
- Persist Tango task, run, and budget state before and after Eino execution.
- Map Eino interrupts to Tango approval or user-input workflows.
- Treat Eino checkpoints as runtime recovery data, not as the source of truth for product state.

### Recommendation

For Tango, Eino is a good fit if the goal is to accelerate agent execution and multi-agent runtime behavior in Go. It is not a replacement for Tango's orchestration, policy, and persistence layers.

---

## Implementation Roadmap

```mermaid
gantt
    title Implementation Roadmap
    dateFormat  YYYY-MM-DD
    section Foundation
    Company + multi-tenant isolation     :p1, 2026-01-01, 7d
    Agent registry + org chart           :p2, after p1, 7d
    Goal tree traversal                  :p3, after p2, 5d

    section Orchestration
    Heartbeat scheduler                  :p4, after p3, 5d
    Atomic executor + budget enforcer    :p5, after p4, 7d
    Agent protocol contract              :p6, after p5, 5d

    section Skill System
    Skill domain + CRUD                  :p7, after p3, 5d
    Versioning + rollback                :p8, after p7, 5d
    Security + content scanner           :p9, after p8, 5d
    Self-evolution proposal flow         :p10, after p9, 7d

    section Context Engine
    Skill resolver                       :p11, after p6, 5d
    Context packer (token management)    :p12, after p11, 5d
    pgvector setup + memory search       :p13, after p12, 7d
    Self-evolution feedback loop         :p14, after p13, 7d
```

### Priority Order

| Phase | Module | Reason |
|---|---|---|
| 1 | Company + Agent Registry | Foundational, everything depends on it |
| 2 | Goal Tree + Org Chart | Required early by the context engine |
| 3 | Heartbeat + Executor + Budget | Core runtime behavior |
| 4 | Skill CRUD + Versioning | Needed before security hardening |
| 5 | Context Engine (packer + skill resolver) | Connects the system together |
| 6 | pgvector + Memory Search | More useful once real data exists |
| 7 | Security + Content Scanner | Hardening and review |
| 8 | Self-Evolution | Best added after production data exists |

---

## Important Design Notes

**Cross-company isolation:** never trust only the application layer. Every relevant query should include `WHERE company_id = $1`. PostgreSQL RLS should act as a second layer of defense.

**Atomic operations:** task checkout and budget deduction must happen inside database transactions. Do not rely on Redis locks or application-level mutexes.

**Skill injection timing:** only inject skills relevant to the current task, never the full skill set. Use tag matching first and semantic similarity second.

**Memory search:** implement this later, once real data exists. Empty embeddings provide little value.

**Self-evolution:** the agent may propose, but the user decides. Never auto-apply proposals. `actor_id` in the audit log should be the `agent_id` so every change remains traceable.
