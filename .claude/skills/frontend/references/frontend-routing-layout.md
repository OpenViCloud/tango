# Frontend Routing And Layout Pattern

Use this reference when working with routing, layouts, shell behavior, or route organization in `web/`.

## Router Structure

- TanStack Router uses file-based routes under `web/src/routes/`
- Generated route tree: `web/src/routeTree.gen.ts`
- Do not hand-edit the generated file

## Layout Routes

- `web/src/routes/__root.tsx`
  - root route
- `web/src/routes/_guest.tsx`
  - guest layout and guest-only flow
- `web/src/routes/_auth.tsx`
  - auth guard + authenticated shell

## Current Route Organization

- File routes stay in `web/src/routes/`
- Page implementations stay in `web/src/pages/`
- Route-specific helpers inside `web/src/routes/` use `-` prefix so the router ignores them

## Shell Rule

- `AppShell` is owned by `web/src/routes/_auth.tsx`
- Auth pages should render content only
- Do not wrap auth pages with `AppShell` directly

## Sidebar/Nav Notes

- Shared sidebar primitives live in `web/src/components/ui/sidebar.tsx`
- Sidebar context hook lives in `web/src/components/ui/sidebar-context.ts`
- Sidebar config lives in `web/src/constants/sidebar.tsx`
- App icons used by nav should come from `web/src/lib/icons.tsx`
