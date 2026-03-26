---
name: frontend
description: "Use when the task is about the frontend in this repository: files under web/, React UI behavior, TanStack Router routes, Zustand auth state, axios client/interceptors, Tailwind v4 styling, Vite build/dev proxy, or integrating the frontend with the Go API."
---

# Frontend

## Overview

This skill covers the `web/` app in the Tango monorepo. Use it for UI changes, route/auth flow fixes, API client wiring, state updates, and frontend build or styling work.

## Project Scope

- App root: `web/`
- Runtime: React 19 + TypeScript + Vite
- Routing: TanStack Router file-based routes under `web/src/routes/`
- Server state: TanStack Query in `web/src/main.tsx`
- Client auth state: Zustand in `web/src/store/auth.ts`
- API client: axios in `web/src/lib/api.ts`
- Styling: Tailwind CSS v4, `@tailwindcss/vite`, `tw-animate-css`
- UI primitives: Radix UI, `class-variance-authority`, `clsx`, `tailwind-merge`, `lucide-react`, `@solar-icons/react`

## Working Rules

1. Start by reading only the frontend files relevant to the request, usually under `web/src/`.
2. Preserve the current architecture instead of introducing a new stack. This app already uses TanStack Router, Zustand, axios, and Tailwind v4.
3. Keep auth flow aligned with the backend contract:
   - `POST /api/auth/login` returns `access_token`
   - refresh token is expected via `httpOnly` cookie
   - refresh happens through `POST /api/auth/refresh`
4. Prefer updating existing patterns over adding abstractions. Do not switch to Redux, React Router, SWR, or another UI kit unless the user explicitly asks.
5. When making API changes on the frontend, verify the matching backend route and payload shape before coding.
6. Follow repo conventions before introducing new abstractions. Prefer the existing pattern of:
   - models in `web/src/@types/models/`
   - API services in `web/src/services/api/`
   - TanStack Query hooks in `web/src/hooks/api/`
   - route-local helpers in `web/src/routes/_auth/<feature>/`
   - page components in `web/src/pages/`
7. For frontend UI work, prefer shadcn/Radix-based components already present in `web/src/components/ui/` and shared helpers in `web/src/components/`.
8. Keep route-specific UI close to the route module. Shared components should stay in `web/src/components/`; feature-only pieces should stay near the route.

## Repo-Specific Notes

- Vite dev server proxies `/api` to `http://localhost:8080` in `web/vite.config.ts`.
- The app bootstraps by attempting `refreshToken()` before rendering routes.
- The generated route tree lives in `web/src/routeTree.gen.ts`; do not hand-edit it.
- Route guards currently live in file routes such as `web/src/routes/_auth.tsx` and `web/src/routes/_guest.tsx`.
- Auth token is intentionally kept in memory, not `localStorage`.
- Production frontend assets are built into `web/dist` and embedded into the Go binary.
- Route helper files under `web/src/routes/` use a `-` prefix when they are not actual route files, so TanStack Router ignores them.
- Shared app icons are centralized in `web/src/lib/icons.tsx`.

## References

Use these reference docs when the task matches the pattern:

- `references/frontend-feature-crud.md`
  - adding or extending a CRUD feature
  - schema -> service -> hook -> route helpers -> page flow
- `references/frontend-routing-layout.md`
  - route structure, `_auth` / `_guest`, shell ownership, sidebar-related route setup
- `references/frontend-data-table.md`
  - TanStack Table list screens, server-side pagination/sorting/search, row selection, bulk actions

## Typical Tasks

- Add or update pages in `web/src/routes/`
- Fix login, logout, refresh-token, or redirect behavior
- Adjust axios interceptors, request headers, or retry logic
- Improve Tailwind styling or shared UI components in `web/src/components/`
- Update Vite config, aliases, or API proxy behavior
- Diagnose frontend build, lint, or typecheck failures

## Conventions

- Data flow:
  - Define Zod schemas and TS types in `web/src/@types/models/<entity>.ts`
  - Add raw API calls in `web/src/services/api/<entity>-service.ts`
  - Add query/mutation hooks in `web/src/hooks/api/use-<entity>.ts`
- Routes:
  - File routes live in `web/src/routes/`
  - Page implementations live in `web/src/pages/`
  - Feature-only columns/forms/helpers should live near the route module
- Forms:
  - Prefer `react-hook-form` + Zod
  - Reuse `web/src/components/form/controlled-field.tsx`
  - Validation messages should use i18n keys rather than raw English fallback strings
- Tables:
  - Prefer TanStack Table for server-side list screens
  - Reuse `web/src/components/data-table/`
  - Keep feature-specific columns and table state in route-local files
- Layout:
  - Authenticated shell is owned by `web/src/routes/_auth.tsx`
  - Pages should render page content, not wrap themselves in `AppShell`
- Icons:
  - Reuse app-level icons from `web/src/lib/icons.tsx`
  - Store icon component types in config instead of pre-rendered JSX when possible
- CRUD UX:
  - Use dedicated pages for larger forms
  - Use `Sheet`/`Dialog` for smaller create/update flows with few fields

## Validation

- Preferred checks:
  - `cd web && pnpm typecheck`
  - `cd web && pnpm build`
  - `cd web && pnpm lint`
- If the task affects auth or routing, also verify the relevant backend route contract before concluding.
