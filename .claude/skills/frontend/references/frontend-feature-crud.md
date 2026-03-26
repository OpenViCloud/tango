# Frontend Feature CRUD Pattern

Use this pattern when adding a new API-backed CRUD feature in `web/`.

## Flow

1. Define schemas and types in `web/src/@types/models/<entity>.ts`
2. Add raw API calls in `web/src/services/api/<entity>-service.ts`
3. Add TanStack Query hooks in `web/src/hooks/api/use-<entity>.ts`
4. Add route-local helpers near the route module:
   - `-columns.tsx`
   - `-use-<entity>-table.ts`
   - `components/-<entity>-form.tsx` or `components/-<entity>-form-sheet.tsx`
5. Compose the screen in `web/src/pages/auth/<feature>/`

## Current Example

- Users:
  - `web/src/@types/models/user.ts`
  - `web/src/services/api/user-service.ts`
  - `web/src/hooks/api/use-user.ts`
  - `web/src/routes/_auth/users/-columns.tsx`
  - `web/src/routes/_auth/users/-use-users-table.ts`
  - `web/src/routes/_auth/users/components/-user-form.tsx`
  - `web/src/pages/auth/users/`
- Roles:
  - `web/src/@types/models/role.ts`
  - `web/src/services/api/role-service.ts`
  - `web/src/hooks/api/use-role.ts`
  - `web/src/routes/_auth/roles/-columns.tsx`
  - `web/src/routes/_auth/roles/-use-roles-table.ts`
  - `web/src/routes/_auth/roles/components/-role-form-sheet.tsx`
  - `web/src/pages/auth/roles/roles-page.tsx`

## Forms

- Prefer `react-hook-form` + Zod
- Reuse `web/src/components/form/controlled-field.tsx`
- Validation messages should use i18n keys
- Use a page for larger forms
- Use a `Sheet` or `Dialog` for small forms with few fields

## UI

- Reuse `PageHeaderCard` and `SectionCard`
- Reuse icons from `web/src/lib/icons.tsx`
- Keep route-specific components near the route, not in shared `components/`
