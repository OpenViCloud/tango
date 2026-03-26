import { Link } from "@tanstack/react-router"
import { useMemo, useState } from "react"

import { DataTable } from "@/components/data-table/data-table"
import { DataTablePagination } from "@/components/data-table/data-table-pagination"
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Button } from "@/components/ui/button"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  InputGroupText,
} from "@/components/ui/input-group"
import { actionIcons, appIcons } from "@/lib/icons"
import { useChannelsTable } from "@/routes/_auth/channels/-use-channels-table"
import { useTranslation } from "react-i18next"

export default function ChannelsListPage() {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const [showFilters, setShowFilters] = useState(false)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)
  const ChannelsIcon = appIcons.channels
  const CreateIcon = actionIcons.create
  const DeleteIcon = actionIcons.delete
  const SearchIcon = actionIcons.search
  const RefreshIcon = actionIcons.refresh
  const FilterIcon = actionIcons.filter
  const SettingsIcon = actionIcons.settings
  const {
    table,
    totalItems,
    isLoading,
    isError,
    isFetching,
    refetch,
    selectedChannelIds,
    handleDeleteSelected,
    deleteChannelsMutation,
    pageSizeOptions,
    sorting,
    setPagination,
    setSorting,
  } = useChannelsTable(query)

  const sortSummary = useMemo(() => {
    const currentSort = sorting[0]

    return t("identity.sort.current", {
      field:
        currentSort?.id === "kind"
          ? t("channels.table.kind")
          : t("channels.table.name"),
      direction: currentSort?.desc
        ? t("identity.sort.descending")
        : t("identity.sort.ascending"),
    })
  }, [sorting, t])

  return (
    <>
      <PageHeaderCard
        icon={<ChannelsIcon />}
        title={t("channels.page.title")}
        description={t("channels.page.description")}
        titleMeta={totalItems}
        headerRight={
          <Button asChild size="lg">
            <Link to="/channels/create">
              <CreateIcon data-icon="inline-start" />
              {t("channels.actions.newChannel")}
            </Link>
          </Button>
        }
      />

      <SectionCard
        headerRight={
          selectedChannelIds.length > 0 && (
            <DeleteConfirmDialog
              open={isDeleteDialogOpen}
              onOpenChange={setIsDeleteDialogOpen}
              title={t("channels.deleteDialog.title")}
              description={t("channels.deleteDialog.description", {
                count: selectedChannelIds.length,
              })}
              confirmLabel={t("channels.deleteDialog.confirm")}
              cancelLabel={t("channels.deleteDialog.cancel")}
              onConfirm={async () => {
                await handleDeleteSelected()
                setIsDeleteDialogOpen(false)
              }}
              trigger={
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  disabled={deleteChannelsMutation.isPending}
                >
                  <DeleteIcon data-icon="inline-start" />
                  {t("channels.actions.deleteSelected", {
                    count: selectedChannelIds.length,
                  })}
                </Button>
              }
            />
          )
        }
      >
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div className="w-full max-w-xl">
              <InputGroup>
                <InputGroupAddon>
                  <InputGroupText>
                    <SearchIcon />
                  </InputGroupText>
                </InputGroupAddon>
                <InputGroupInput
                  value={query}
                  placeholder={t("channels.searchPlaceholder")}
                  onChange={(event) => {
                    setQuery(event.target.value)
                    setPagination((current) => ({ ...current, pageIndex: 0 }))
                  }}
                />
              </InputGroup>
            </div>

            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={isFetching}
                onClick={() => {
                  void refetch()
                }}
              >
                <RefreshIcon data-icon="inline-start" />
                {t("identity.toolbar.refresh")}
              </Button>

              <Collapsible open={showFilters} onOpenChange={setShowFilters}>
                <CollapsibleTrigger asChild>
                  <Button type="button" variant="outline" size="sm">
                    <FilterIcon data-icon="inline-start" />
                    {t("identity.toolbar.filters")}
                  </Button>
                </CollapsibleTrigger>
              </Collapsible>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button type="button" variant="outline" size="sm">
                    <SettingsIcon data-icon="inline-start" />
                    {t("identity.toolbar.columns")}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-44">
                  <DropdownMenuLabel>
                    {t("identity.toolbar.toggleColumns")}
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  {table
                    .getAllColumns()
                    .filter((column) => column.getCanHide())
                    .map((column) => (
                      <DropdownMenuCheckboxItem
                        key={column.id}
                        checked={column.getIsVisible()}
                        onCheckedChange={(value) =>
                          column.toggleVisibility(!!value)
                        }
                      >
                        {column.id === "name"
                          ? t("channels.table.name")
                          : column.id === "kind"
                            ? t("channels.table.kind")
                            : t("channels.table.status")}
                      </DropdownMenuCheckboxItem>
                    ))}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>

          <Collapsible open={showFilters} onOpenChange={setShowFilters}>
            <CollapsibleContent>
              <div className="grid gap-3 rounded-2xl border bg-muted/20 p-4 md:grid-cols-2">
                <div className="flex flex-col gap-2">
                  <p className="text-sm font-medium">{t("channels.filters.sortBy")}</p>
                  <div className="flex flex-wrap gap-2">
                    <Button
                      type="button"
                      size="sm"
                      variant={
                        sorting[0]?.id === "name" ? "secondary" : "outline"
                      }
                      onClick={() => setSorting([{ id: "name", desc: false }])}
                    >
                      {t("channels.table.name")}
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant={
                        sorting[0]?.id === "kind" ? "secondary" : "outline"
                      }
                      onClick={() => setSorting([{ id: "kind", desc: false }])}
                    >
                      {t("channels.table.kind")}
                    </Button>
                  </div>
                </div>

                <div className="flex flex-col gap-2">
                  <p className="text-sm font-medium">
                    {t("identity.filters.direction")}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <Button
                      type="button"
                      size="sm"
                      variant={sorting[0]?.desc ? "outline" : "secondary"}
                      onClick={() =>
                        setSorting((current) => [
                          { id: current[0]?.id ?? "name", desc: false },
                        ])
                      }
                    >
                      {t("identity.sort.ascending")}
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant={sorting[0]?.desc ? "secondary" : "outline"}
                      onClick={() =>
                        setSorting((current) => [
                          { id: current[0]?.id ?? "name", desc: true },
                        ])
                      }
                    >
                      {t("identity.sort.descending")}
                    </Button>
                  </div>
                </div>
              </div>
            </CollapsibleContent>
          </Collapsible>

          <div className="flex items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">{sortSummary}</p>
          </div>

          <DataTable
            table={table}
            loading={isLoading}
            emptyMessage={
              isError
                ? t("channels.errors.loadChannels")
                : t("channels.emptyChannels")
            }
            skeletonRows={5}
          />

          <DataTablePagination
            table={table}
            rowCount={totalItems}
            isFetching={isFetching}
            pageSizeOptions={pageSizeOptions}
            selectedCount={selectedChannelIds.length}
          />
        </div>
      </SectionCard>
    </>
  )
}
