import { useEffect, useState } from "react"
import { type RowData, type Table as TanStackTable } from "@tanstack/react-table"
import { ChevronLeftIcon, ChevronRightIcon, ChevronsUpDownIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { useTranslation } from "react-i18next"

type DataTablePaginationProps<TData extends RowData> = {
  table: TanStackTable<TData>
  rowCount: number
  isFetching?: boolean
  pageSizeOptions: readonly number[]
  selectedCount?: number
}

export function DataTablePagination<TData extends RowData>({
  table,
  rowCount,
  isFetching = false,
  pageSizeOptions,
  selectedCount = 0,
}: DataTablePaginationProps<TData>) {
  const { t } = useTranslation()
  const { pageIndex, pageSize } = table.getState().pagination
  const currentPage = pageIndex + 1
  const pageCount = table.getPageCount()
  const rangeStart = rowCount === 0 ? 0 : pageIndex * pageSize + 1
  const rangeEnd = rowCount === 0 ? 0 : Math.min((pageIndex + 1) * pageSize, rowCount)
  const [pageInput, setPageInput] = useState(String(currentPage))

  useEffect(() => {
    setPageInput(String(currentPage))
  }, [currentPage])

  const jumpToPage = () => {
    const parsed = Number(pageInput)

    if (!Number.isInteger(parsed) || pageCount === 0) {
      setPageInput(String(currentPage))
      return
    }

    const clampedPage = Math.min(Math.max(parsed, 1), pageCount)
    table.setPageIndex(clampedPage - 1)
    setPageInput(String(clampedPage))
  }

  return (
    <div className="mt-4 flex flex-col gap-3 border-t pt-4 md:flex-row md:items-center md:justify-end">
      <p className="text-sm text-muted-foreground">
        {selectedCount > 0
          ? t("identity.pagination.selected", { count: selectedCount })
          : t("identity.pagination.rangeSummary", {
              from: rangeStart,
              to: rangeEnd,
              totalItems: rowCount,
            })}
        {isFetching ? ` • ${t("identity.pagination.refreshing")}` : ""}
      </p>

      <div className="flex items-center gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline" size="sm">
              {t("identity.pagination.perPage", { count: pageSize })}
              <ChevronsUpDownIcon data-icon="inline-end" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-40">
            {pageSizeOptions.map((option) => (
              <DropdownMenuCheckboxItem
                key={option}
                checked={pageSize === option}
                onCheckedChange={() => {
                  table.setPageSize(option)
                  table.setPageIndex(0)
                }}
              >
                {t("identity.pagination.perPage", { count: option })}
              </DropdownMenuCheckboxItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <span>{t("identity.pagination.page")}</span>
          <Input
            value={pageInput}
            inputMode="numeric"
            className="h-9 w-16 text-center"
            onChange={(event) => setPageInput(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault()
                jumpToPage()
              }
            }}
            onBlur={jumpToPage}
          />
          <span>{t("identity.pagination.of", { totalPage: Math.max(pageCount, 1) })}</span>
        </div>

        <Button
          type="button"
          size="icon-sm"
          variant="outline"
          disabled={!table.getCanPreviousPage() || isFetching}
          onClick={() => table.previousPage()}
        >
          <ChevronLeftIcon />
        </Button>
        <Button
          type="button"
          size="icon-sm"
          variant="outline"
          disabled={!table.getCanNextPage() || isFetching}
          onClick={() => table.nextPage()}
        >
          <ChevronRightIcon />
        </Button>
      </div>
    </div>
  )
}
