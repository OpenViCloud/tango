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
import { useUsersTable } from "@/routes/_auth/users/-use-users-table"
import { useTranslation } from "react-i18next"

export default function UsersListPage() {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const [showFilters, setShowFilters] = useState(false)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)
  const UsersIcon = appIcons.users
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
    selectedUserIds,
    handleDeleteSelected,
    deleteUsersMutation,
    pageSizeOptions,
    sorting,
    setPagination,
    setSorting,
  } = useUsersTable(query)

  const sortSummary = useMemo(() => {
    const currentSort = sorting[0]

    return t("identity.sort.current", {
      field:
        currentSort?.id === "first_name"
          ? t("identity.table.name")
          : t("identity.table.email"),
      direction: currentSort?.desc
        ? t("identity.sort.descending")
        : t("identity.sort.ascending"),
    })
  }, [sorting, t])

  return (
    <>
      <PageHeaderCard
        icon={<UsersIcon />}
        title={t("identity.page.title")}
        description={t("identity.page.description")}
        titleMeta={totalItems}
        headerRight={
          <Button asChild size="lg">
            <Link to="/users/create">
              <CreateIcon data-icon="inline-start" />
              {t("identity.actions.newUser")}
            </Link>
          </Button>
        }
      />

      <SectionCard
        headerRight={
          selectedUserIds.length > 0 && (
            <DeleteConfirmDialog
              open={isDeleteDialogOpen}
              onOpenChange={setIsDeleteDialogOpen}
              title={t("identity.deleteDialog.title")}
              description={t("identity.deleteDialog.description", {
                count: selectedUserIds.length,
              })}
              confirmLabel={t("identity.deleteDialog.confirm")}
              cancelLabel={t("identity.deleteDialog.cancel")}
              onConfirm={async () => {
                await handleDeleteSelected()
                setIsDeleteDialogOpen(false)
              }}
              trigger={
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  disabled={deleteUsersMutation.isPending}
                >
                  <DeleteIcon data-icon="inline-start" />
                  {t("identity.actions.deleteSelected", {
                    count: selectedUserIds.length,
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
                  placeholder={t("identity.searchPlaceholder")}
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
                        {column.id === "first_name"
                          ? t("identity.table.name")
                          : column.id === "email"
                            ? t("identity.table.email")
                            : column.id === "phone"
                              ? t("identity.table.phone")
                              : t("identity.table.status")}
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
                  <p className="text-sm font-medium">
                    {t("identity.filters.sortBy")}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <Button
                      type="button"
                      size="sm"
                      variant={
                        sorting[0]?.id === "first_name"
                          ? "secondary"
                          : "outline"
                      }
                      onClick={() =>
                        setSorting([{ id: "first_name", desc: false }])
                      }
                    >
                      {t("identity.table.name")}
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant={
                        sorting[0]?.id === "email" ? "secondary" : "outline"
                      }
                      onClick={() => setSorting([{ id: "email", desc: false }])}
                    >
                      {t("identity.table.email")}
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
                          { id: current[0]?.id ?? "email", desc: false },
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
                          { id: current[0]?.id ?? "email", desc: true },
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
            <span className="text-sm text-muted-foreground">
              {t("identity.pagination.summary", {
                currentPage: table.getState().pagination.pageIndex + 1,
                totalPage: Math.max(table.getPageCount(), 1),
                totalItems,
              })}
            </span>
          </div>

          {isError ? (
            <div className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
              {t("identity.errors.loadUsers")}
            </div>
          ) : (
            <>
              <DataTable
                table={table}
                loading={isLoading}
                emptyMessage={t("identity.emptyUsers")}
              />
              <DataTablePagination
                table={table}
                rowCount={totalItems}
                isFetching={isFetching && !isLoading}
                pageSizeOptions={pageSizeOptions}
                selectedCount={selectedUserIds.length}
              />
            </>
          )}
        </div>
      </SectionCard>
    </>
  )
}
