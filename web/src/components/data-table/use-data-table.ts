import {
  getCoreRowModel,
  type RowData,
  useReactTable,
  type TableOptions,
} from "@tanstack/react-table"

export function useDataTable<TData extends RowData>(
  options: Omit<TableOptions<TData>, "getCoreRowModel">
) {
  return useReactTable({
    ...options,
    getCoreRowModel: getCoreRowModel(),
  })
}
