# Frontend Data Table Pattern

Use this reference when building list screens with search, sorting, pagination, bulk actions, or column visibility.

## Shared Table Layer

- `web/src/components/data-table/use-data-table.ts`
- `web/src/components/data-table/data-table.tsx`
- `web/src/components/data-table/data-table-pagination.tsx`

These files provide the generic TanStack Table wrapper.

## Feature-Specific Table Layer

Keep feature table logic near the route:

- `-columns.tsx`
  - feature-specific column definitions
- `-use-<feature>-table.ts`
  - pagination state
  - sorting state
  - row selection state
  - API mapping
  - bulk actions

## Current Examples

- Users:
  - `web/src/routes/_auth/users/-columns.tsx`
  - `web/src/routes/_auth/users/-use-users-table.ts`
- Roles:
  - `web/src/routes/_auth/roles/-columns.tsx`
  - `web/src/routes/_auth/roles/-use-roles-table.ts`

## Preferred Behavior

- Prefer server-side search, sorting, and pagination when the API supports them
- Keep fetch logic out of the generic `DataTable`
- Use page-level toolbars for:
  - search
  - refresh
  - filter collapse
  - column visibility
  - bulk actions

## Selection

- Bulk delete should be tied to row selection
- If the backend exposes protected records, disable selection per row instead of rendering a special list

## Styling

- Let the surrounding page/card control container styling
- Keep the shared table generic and presentation-light
