import type { ColumnDef, SortingState } from "@tanstack/react-table"
import {
  ArrowDownAZIcon,
  ArrowUpAZIcon,
  ChevronsUpDownIcon,
} from "lucide-react"
import { Link } from "@tanstack/react-router"

import type { ChannelModel } from "@/@types/models"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { actionIcons } from "@/lib/icons"
import type { TFunction } from "i18next"

function statusVariant(
  status: ChannelModel["status"]
): "success" | "warning" | "outline" {
  if (status === "active") return "success"
  if (status === "pending") return "warning"
  return "outline"
}

type ChannelColumnsOptions = {
  t: TFunction
  sorting: SortingState
  onStart: (channelId: string) => void
  onStop: (channelId: string) => void
  onRestart: (channelId: string) => void
  actionPending?: boolean
}

export function getChannelColumns({
  t,
  sorting,
  onStart,
  onStop,
  onRestart,
  actionPending = false,
}: ChannelColumnsOptions): ColumnDef<ChannelModel>[] {
  const nameSorted = sorting[0]?.id === "name" ? sorting[0] : null
  const StartIcon = actionIcons.start
  const StopIcon = actionIcons.stop
  const RestartIcon = actionIcons.restart

  return [
    {
      id: "select",
      enableSorting: false,
      enableHiding: false,
      header: ({ table }) => (
        <Checkbox
          aria-label={t("channels.table.selectAll")}
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(checked) =>
            table.toggleAllPageRowsSelected(checked === true)
          }
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          aria-label={t("channels.table.selectRow", { name: row.original.name })}
          checked={row.getIsSelected()}
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
          {t("channels.table.name")}
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
      accessorKey: "kind",
      header: () => t("channels.table.kind"),
      cell: ({ row }) => row.original.kind,
    },
    {
      accessorKey: "status",
      header: () => t("channels.table.status"),
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
      header: () => <div className="text-right">{t("channels.table.actions")}</div>,
      cell: ({ row }) => {
        const canRun = row.original.has_credentials
        const isActive = row.original.status === "active"

        return (
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={!canRun || isActive || actionPending}
              onClick={() => onStart(row.original.id)}
            >
              <StartIcon data-icon="inline-start" />
              {t("channels.actions.start")}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={!canRun || !isActive || actionPending}
              onClick={() => onStop(row.original.id)}
            >
              <StopIcon data-icon="inline-start" />
              {t("channels.actions.stop")}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={!canRun || actionPending}
              onClick={() => onRestart(row.original.id)}
            >
              <RestartIcon data-icon="inline-start" />
              {t("channels.actions.restart")}
            </Button>
            <Button asChild size="sm" variant="outline">
              <Link to="/channels/$channelId/edit" params={{ channelId: row.original.id }}>
                {t("identity.actions.edit")}
              </Link>
            </Button>
          </div>
        )
      },
    },
  ]
}
