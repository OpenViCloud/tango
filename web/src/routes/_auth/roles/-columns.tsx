import type { ColumnDef, SortingState } from "@tanstack/react-table"
import {
  ArrowDownAZIcon,
  ArrowUpAZIcon,
  ChevronsUpDownIcon,
} from "lucide-react"

import type { RoleModel } from "@/@types/models"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import type { TFunction } from "i18next"

type RoleColumnsOptions = {
  t: TFunction
  sorting: SortingState
  onEdit: (role: RoleModel) => void
}

export function getRoleColumns({
  t,
  sorting,
  onEdit,
}: RoleColumnsOptions): ColumnDef<RoleModel>[] {
  const nameSorted = sorting[0]?.id === "name" ? sorting[0] : null

  return [
    {
      id: "select",
      enableSorting: false,
      enableHiding: false,
      header: ({ table }) => (
        <Checkbox
          aria-label={t("identity.roles.table.selectAll")}
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(checked) =>
            table.toggleAllPageRowsSelected(checked === true)
          }
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          aria-label={t("identity.roles.table.selectRow", {
            name: row.original.name,
          })}
          checked={row.getIsSelected()}
          disabled={row.original.is_system}
          onCheckedChange={(checked) => row.toggleSelected(checked === true)}
        />
      ),
    },
    {
      accessorKey: "name",
      header: ({ column }) => (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="-ml-2 h-8 px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground hover:bg-transparent hover:text-foreground"
          onClick={() => column.toggleSorting(nameSorted?.desc === false)}
        >
          {t("identity.roles.table.name")}
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
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
    },
    {
      accessorKey: "description",
      header: () => t("identity.roles.table.description"),
      cell: ({ row }) => row.original.description || "-",
    },
    {
      accessorKey: "updated_at",
      header: () => t("identity.roles.table.updatedAt"),
      cell: ({ row }) =>
        new Intl.DateTimeFormat(undefined, {
          dateStyle: "medium",
          timeStyle: "short",
        }).format(new Date(row.original.updated_at)),
    },
    {
      id: "actions",
      enableSorting: false,
      enableHiding: false,
      header: () => (
        <div className="text-right">{t("identity.roles.table.actions")}</div>
      ),
      cell: ({ row }) => (
        <div className="text-right">
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => onEdit(row.original)}
          >
            {t("identity.actions.edit")}
          </Button>
        </div>
      ),
    },
  ]
}
