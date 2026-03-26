import type { ColumnDef, SortingState } from "@tanstack/react-table"
import {
  ArrowDownAZIcon,
  ArrowUpAZIcon,
  ChevronsUpDownIcon,
  SquarePenIcon,
} from "lucide-react"
import { Link } from "@tanstack/react-router"

import type { UserModel } from "@/@types/models"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import type { TFunction } from "i18next"

function statusVariant(status: string): "success" | "warning" | "outline" {
  if (status === "active") return "success"
  if (status === "inactive") return "warning"
  return "outline"
}

type UserColumnsOptions = {
  t: TFunction
  sorting: SortingState
}

export function getUserColumns({
  t,
  sorting,
}: UserColumnsOptions): ColumnDef<UserModel>[] {
  const nameSorted = sorting[0]?.id === "first_name" ? sorting[0] : null
  const emailSorted = sorting[0]?.id === "email" ? sorting[0] : null

  return [
    {
      id: "select",
      enableSorting: false,
      enableHiding: false,
      header: ({ table }) => (
        <Checkbox
          aria-label={t("identity.table.selectAll")}
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(checked) =>
            table.toggleAllPageRowsSelected(checked === true)
          }
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          aria-label={t("identity.table.selectRow", {
            name: `${row.original.first_name} ${row.original.last_name}`.trim(),
          })}
          checked={row.getIsSelected()}
          onCheckedChange={(checked) => row.toggleSelected(checked === true)}
        />
      ),
    },
    {
      accessorKey: "first_name",
      header: ({ column }) => (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="-ml-2 h-8 px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground hover:bg-transparent hover:text-foreground"
          onClick={() => column.toggleSorting(nameSorted?.desc === false)}
        >
          {t("identity.table.name")}
          {nameSorted ? (
            nameSorted.desc ? (
              <ArrowDownAZIcon data-icon="inline-end" />
            ) : (
              <ArrowUpAZIcon data-icon="inline-end" />
            )
          ) : (
            <ChevronsUpDownIcon data-icon="inline-end" />
          )}
        </Button>
      ),
      cell: ({ row }) => (
        <span className="font-medium">
          {row.original.first_name} {row.original.last_name}
        </span>
      ),
    },
    {
      accessorKey: "email",
      header: ({ column }) => (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="-ml-2 h-8 px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground hover:bg-transparent hover:text-foreground"
          onClick={() => column.toggleSorting(emailSorted?.desc === false)}
        >
          {t("identity.table.email")}
          {emailSorted ? (
            emailSorted.desc ? (
              <ArrowDownAZIcon data-icon="inline-end" />
            ) : (
              <ArrowUpAZIcon data-icon="inline-end" />
            )
          ) : (
            <ChevronsUpDownIcon data-icon="inline-end" />
          )}
        </Button>
      ),
    },
    {
      accessorKey: "phone",
      header: () => t("identity.table.phone"),
      cell: ({ row }) => row.original.phone || "-",
    },
    {
      accessorKey: "status",
      header: () => t("identity.table.status"),
      cell: ({ row }) => (
        <Badge variant={statusVariant(row.original.status)}>
          {row.original.status}
        </Badge>
      ),
    },
    {
      id: "actions",
      enableSorting: false,
      enableHiding: false,
      header: () => (
        <div className="text-right">{t("identity.table.actions")}</div>
      ),
      cell: ({ row }) => (
        <div className="text-right">
          <Button asChild size="sm" variant="outline">
            <Link to="/users/$userId/edit" params={{ userId: row.original.id }}>
              <SquarePenIcon data-icon="inline-start" />
              {t("identity.actions.edit")}
            </Link>
          </Button>
        </div>
      ),
    },
  ]
}
