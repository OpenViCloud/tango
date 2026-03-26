# Tango MVP Scope

> This document defines the pragmatic MVP cut for Tango. It is intentionally narrower than the long-term target architecture in `roadmap/ARCHITECTURE.md`.

---

## Goal

The MVP should be enough to ship a usable Tango foundation with:

- workspace management
- agent configuration
- chat and conversation history
- Kanban-style task board
- task movement across stages
- basic task history

This is the smallest scope that still feels like a real Tango product.

---

## Phase 1 Foundation

Phase 1 defines the workspace-centric foundation that should be completed first:

- `users`
- `roles`
- `user_roles`
- `workspaces`
- `workspace_members`
- `llm_providers`
- `agents`
- `channels`
- `conversations`
- `conversation_messages`
- `pipelines`
- `pipeline_stages`
- `tasks`
- `task_stage_history`

---

## Pre-Implementation Checklist

Before implementation starts, the Phase 1 schema work should be split into three buckets: update existing domains, add new tables, and defer later domains.

### Update Existing Domains

- `workspaces`
  - keep as the root container for project-level execution
  - retain `name`, `description`, `status`, and `metadata_json`
- `llm_providers`
  - treat as provider records that contain model config and encrypted API keys
- `agents`
  - use `agent_providers` for primary + fallback provider routing
  - retain `workspace_id`, `parent_agent_id`, `role`, `type`, `status`, `system_prompt`, `model_override`, `temperature`, `max_tokens`, and `metadata_json`
- `channels`
  - keep `workspace_id`
  - treat each channel as owned by exactly one workspace
- `conversations`
  - add `workspace_id`
  - replace `channel_kind` with `channel_id`
  - add `conversation_type`
  - keep `user_id` nullable
- `conversation_messages`
  - add `sender_type`
  - add `sender_id`
  - keep `role` as the transcript role, not the sender identity

### Add New Tables

- `workspace_members`
  - `id`
  - `workspace_id`
  - `user_id`
  - `member_role`
  - `created_at`
- `agent_providers`
  - `id`
  - `agent_id`
  - `llm_provider_id`
  - `priority`
  - `created_at`
  - optional `updated_at`
  - optional `status`

### Field Changes

- `agents.llm_provider_id` -> `agent_providers.llm_provider_id`
- `conversations.channel_kind` -> `conversations.channel_id`
- add `conversations.workspace_id`
- add `conversations.conversation_type`
- add `conversation_messages.sender_type`
- add `conversation_messages.sender_id`

### Keep As-Is For Now

- `users`
- `roles`
- `user_roles`

### Defer Until Later Phases

- `skills`
- `agent_skills`
- `knowledge_sources`
- `agent_knowledge_sources`
- `pipelines`
- `pipeline_stages`
- `tasks`
- `task_stage_history`
- `pipeline_runs`
- `pipeline_run_steps`

### Recommended Order

1. Clean up `llm_providers`
2. Add `agent_providers`
3. Update `agents`
4. Add `workspace_members`
5. Update `channels`
6. Update `conversations`
7. Update `conversation_messages`

---

## Phase 2 Deferred After MVP

These tables/modules are intentionally deferred until after the MVP foundation is stable:

- `skills`
- `agent_skills`
- `knowledge_sources`
- `agent_knowledge_sources`
- `workflows`
- `workflow_nodes`
- `workflow_edges`
- `runs`
- `run_steps`

### Planned Multi-Agent Execution Schema

The post-MVP multi-agent layer should separate hierarchy, workflow definition, and execution history:

- `agents`
  - keep `workspace_id`
  - keep `kind`
  - keep `parent_agent_id`
  - use this for supervisor tree and delegation hierarchy
- `workflows`
  - `id`
  - `workspace_id`
  - `name`
  - `description`
  - `status`
  - `trigger_mode`
  - `created_at`
  - `updated_at`
- `workflow_nodes`
  - `id`
  - `workflow_id`
  - `agent_id`
  - `name`
  - `node_type`
  - `position_x`
  - `position_y`
  - `config_json`
  - `created_at`
- `workflow_edges`
  - `id`
  - `workflow_id`
  - `from_node_id`
  - `to_node_id`
  - `execution_mode`
  - `label`
  - `created_at`
- `runs`
  - `id`
  - `workspace_id`
  - `workflow_id` nullable
  - `conversation_id` nullable
  - `entry_agent_id`
  - `status`
  - `input`
  - `final_output`
  - `started_at`
  - `finished_at`
- `run_steps`
  - `id`
  - `run_id`
  - `workflow_node_id` nullable
  - `agent_id`
  - `step_index`
  - `step_type`
  - `status`
  - `input`
  - `output`
  - `metadata_json`
  - `started_at`
  - `finished_at`

### Planned Multi-Agent Services

- `OrchestrationService`
  - entrypoint selection
  - supervisor tree orchestration
  - single-agent fallback path
- `WorkflowService`
  - CRUD and validation for workflow, nodes, and edges
  - data source for future graph UI
- `WorkflowExecutionService`
  - load DAG definitions from DB
  - execute sequential and parallel graph runs
  - future integration point for Eino graph runtime
- `RunTraceService`
  - persist `runs` and `run_steps`
  - power execution trace and debug views

---

## Why This Cut Is Enough

This Phase 1 scope is enough to support the core Tango loop:

1. A user belongs to a workspace and can collaborate with others.
2. A workspace can configure LLM providers, agents, and channels.
3. Agents can participate in conversations and retain message history.
4. Work can be organized into pipelines and stages.
5. Tasks can move across stages and retain a basic audit trail through `task_stage_history`.

This gives Tango a coherent MVP around operations, collaboration, and execution flow without pulling in more advanced systems too early.

---

## Explicit Non-Goals For MVP

The MVP does not attempt to fully implement:

- skill learning and assignment
- knowledge source ingestion and retrieval
- graph workflow definition and execution
- execution run tracing and step persistence
- the full long-term autonomous orchestration layer described in the target architecture

Those belong to later phases after the Phase 1 workspace, provider, agent, chat, and task flows are stable.
