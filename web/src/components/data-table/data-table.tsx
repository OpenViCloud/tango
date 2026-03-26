import { flexRender, type RowData, type Table as TanStackTable } from "@tanstack/react-table"

import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

type DataTableProps<TData extends RowData> = {
  table: TanStackTable<TData>
  loading?: boolean
  emptyMessage: string
  skeletonRows?: number
}

export function DataTable<TData extends RowData>({
  table,
  loading = false,
  emptyMessage,
  skeletonRows = 3,
}: DataTableProps<TData>) {
  const columnCount = table.getAllLeafColumns().length

  return (
    <Table>
      <TableHeader>
        {table.getHeaderGroups().map((headerGroup) => (
          <TableRow key={headerGroup.id}>
            {headerGroup.headers.map((header) => (
              <TableHead key={header.id}>
                {header.isPlaceholder
                  ? null
                  : flexRender(
                      header.column.columnDef.header,
                      header.getContext()
                    )}
              </TableHead>
            ))}
          </TableRow>
        ))}
      </TableHeader>
      <TableBody>
        {loading
          ? Array.from({ length: skeletonRows }).map((_, index) => (
              <TableRow key={`skeleton-${index}`}>
                <TableCell colSpan={columnCount}>
                  <Skeleton className="h-10 w-full" />
                </TableCell>
              </TableRow>
            ))
          : table.getRowModel().rows.map((row) => (
              <TableRow
                key={row.id}
                data-state={row.getIsSelected() ? "selected" : undefined}
              >
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))}

        {!loading && table.getRowModel().rows.length === 0 ? (
          <TableRow>
            <TableCell
              colSpan={columnCount}
              className="py-10 text-center text-muted-foreground"
            >
              {emptyMessage}
            </TableCell>
          </TableRow>
        ) : null}
      </TableBody>
    </Table>
  )
}
