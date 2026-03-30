# AGENTS.md instructions for /Users/felix/project-repos/tango-cloud

## Skills

A skill is a set of local instructions stored in a `SKILL.md` file.

### Available skills

This repository currently does not check in additional project-local `SKILL.md` files.

If session-level skills are available, use them as follows:

- frontend: Use when the task is about the frontend in `web/`, including TanStack Router routes, auth state, axios clients, Tailwind styling, settings/pages for resources, domains, builds, and sources.
- backend: Use when the task is about the backend in `cmd/` or `internal/`, including Gin routes, JWT auth, Docker runtime, BuildKit, source connections, settings, domain routing, Traefik integration, and frontend asset embedding.
- shadcn: Use when frontend work should follow shadcn/ui design and component patterns, including adding missing UI components before building a page.

### How to use skills

- Discovery: Prefer repo-local skills if they are added later under `.codex/skills/` or `.agents/skills/`. Otherwise rely on session-provided skills.
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
  - Prefer the backend skill for changes contained to `cmd/`, `internal/`, Docker, Traefik, or API behavior.
  - Avoid pulling both skills into context unless the task truly crosses the FE/BE boundary.
