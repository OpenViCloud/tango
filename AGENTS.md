# AGENTS.md instructions for /Users/felix/projects/tango

## Skills

A skill is a set of local instructions stored in a `SKILL.md` file. The following project-local skills are available in this repository.

### Available skills

- frontend: Use when the task is about the frontend in `web/`, including React UI behavior, TanStack Router routes, Zustand auth state, axios client/interceptors, Tailwind styling, or Vite configuration. (file: /Users/felix/projects/tango/.codex/skills/frontend/SKILL.md)
- backend: Use when the task is about the backend in `cmd/` or `internal/`, including Gin routes, JWT auth, bcrypt, SSE, CLI commands, config, Docker build, or frontend asset embedding. (file: /Users/felix/projects/tango/.codex/skills/backend/SKILL.md)
- shadcn: Use when frontend work should follow shadcn/ui design and component patterns, including adding missing UI components or setting up shadcn before building the page. (file: /Users/felix/projects/tango/.codex/skills/shadcn/SKILL.md)

### How to use skills

- Discovery: The list above is the skills available in this repository.
- Trigger rules: If the user names a skill with `$frontend`, `$backend`, or `$shadcn`, or the task clearly matches one description, use that skill for the turn. If a task spans both UI and API contracts, use both skills but keep frontend and backend reasoning separated.
- Missing/blocked: If a listed skill cannot be read, say so briefly and continue with normal repo inspection.
- Progressive disclosure:
  1. Open the referenced `SKILL.md`.
  2. Read only the parts needed for the current task.
  3. Load extra files from the repo only when the task requires them.
- Context hygiene:
  - Prefer the frontend skill for changes contained to `web/`.
  - For frontend UI work, also use the `shadcn` skill by default and follow shadcn design/component patterns.
  - If the required shadcn component does not exist yet, add it before building the page.
  - If shadcn is not set up yet for the frontend, set it up first, then continue with the UI task.
  - Prefer the backend skill for changes contained to `cmd/`, `internal/`, Docker, or API/CLI behavior.
  - Avoid pulling both skills into context unless the task truly crosses the FE/BE boundary.
