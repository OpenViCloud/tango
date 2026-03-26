# Frontend Conventions

This document captures the working conventions used in `web/`.

## Architecture

- Routing: TanStack Router file-based routes in `src/routes/`
- Page implementations: `src/pages/`
- Shared UI: `src/components/`
- Route-specific helpers: keep them close to the route module, usually under `src/routes/_auth/<feature>/`

## Feature Structure

For a new API-backed entity, prefer this flow:

1. `src/@types/models/<entity>.ts`
   - Zod schemas
   - inferred request/response types
   - list request params when needed
2. `src/services/api/<entity>-service.ts`
   - raw API calls only
   - unwrap API response shape here
3. `src/hooks/api/use-<entity>.ts`
   - TanStack Query list/detail hooks
   - create/update/delete mutations
   - cache invalidation
4. `src/routes/_auth/<feature>/`
   - feature table columns
   - feature table state hook
   - feature-only form or sheet components
5. `src/pages/auth/<feature>/`
   - page composition and screen-level behavior

## Routes

- Actual route files belong in `src/routes/`
- Page content belongs in `src/pages/`
- Helper files inside `src/routes/` that are not routes should use a `-` prefix
- Do not hand-edit `src/routeTree.gen.ts`

## Layout

- The authenticated shell is defined in `src/routes/_auth.tsx`
- Guest layout is defined in `src/routes/_guest.tsx`
- Auth pages should not wrap themselves with `AppShell`
- Pages should render content only: header, cards, tables, forms

## Forms

- Prefer `react-hook-form` with Zod
- Reuse `src/components/form/controlled-field.tsx`
- Keep larger forms in dedicated pages
- Keep small create/update flows in a `Sheet` or `Dialog`
- Validation errors should come from i18n keys, not raw hardcoded English strings

## Tables

- Prefer TanStack Table for list screens
- Reuse:
  - `src/components/data-table/data-table.tsx`
  - `src/components/data-table/data-table-pagination.tsx`
  - `src/components/data-table/use-data-table.ts`
- Keep feature-specific column definitions and table state in route-local files
- Prefer server-side search, pagination, and sorting when the API supports them

## Icons

- Shared app icons live in `src/lib/icons.tsx`
- Reuse the same domain icon across sidebar and related pages
- Store icon component types in config instead of pre-rendered JSX when possible
- Pass `className`, `strokeWidth`, and similar props at render sites

## UI Composition

- Use shared cards for consistent page structure:
  - `src/components/share/cards/page-header-card.tsx`
  - `src/components/share/cards/section-card.tsx`
- Use shadcn/Radix primitives already present in `src/components/ui/`
- Prefer extending an existing pattern over introducing a parallel one

## CRUD UX

- Use page-based create/edit when the form is larger or needs richer context
- Use sheet-based create/edit when there are only a few fields
- Bulk actions should be tied to table row selection
- Global mutation errors should go through toast unless a mutation explicitly opts out

## Validation

Run these after frontend changes:

```bash
cd web
pnpm typecheck
pnpm build
```
