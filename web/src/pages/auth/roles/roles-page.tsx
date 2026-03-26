import { useMemo, useState } from "react"

import {
  createRoleSchema,
  type CreateRoleModel,
  type RoleModel,
  updateRoleSchema,
} from "@/@types/models"
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
import { useCreateRole, useUpdateRole } from "@/hooks/api/use-role"
import { actionIcons, appIcons } from "@/lib/icons"
import { useRolesTable } from "@/routes/_auth/roles/-use-roles-table"
import { RoleFormSheet } from "@/routes/_auth/roles/components/-role-form-sheet"
import { useTranslation } from "react-i18next"

export default function RolesPage() {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const [showFilters, setShowFilters] = useState(false)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)
  const [sheetMode, setSheetMode] = useState<"create" | "update">("create")
  const [isSheetOpen, setIsSheetOpen] = useState(false)
  const [selectedRole, setSelectedRole] = useState<RoleModel | null>(null)
  const RolesIcon = appIcons.roles
  const CreateIcon = actionIcons.create
  const DeleteIcon = actionIcons.delete
  const SearchIcon = actionIcons.search
  const RefreshIcon = actionIcons.refresh
  const FilterIcon = actionIcons.filter
  const SettingsIcon = actionIcons.settings
  const createRoleMutation = useCreateRole()
  const updateRoleMutation = useUpdateRole()

  const handleOpenCreate = () => {
    setSheetMode("create")
    setSelectedRole(null)
    setIsSheetOpen(true)
  }

  const handleOpenEdit = (role: RoleModel) => {
    setSheetMode("update")
    setSelectedRole(role)
    setIsSheetOpen(true)
  }

  const {
    table,
    totalItems,
    isLoading,
    isError,
    isFetching,
    refetch,
    selectedRoleIds,
    handleDeleteSelected,
    deleteRolesMutation,
    pageSizeOptions,
    sorting,
    setPagination,
    setSorting,
  } = useRolesTable(query, handleOpenEdit)

  const sortSummary = useMemo(() => {
    const currentSort = sorting[0]

    return t("identity.sort.current", {
      field:
        currentSort?.id === "updated_at"
          ? t("identity.roles.table.updatedAt")
          : t("identity.roles.table.name"),
      direction: currentSort?.desc
        ? t("identity.sort.descending")
        : t("identity.sort.ascending"),
    })
  }, [sorting, t])

  const handleSubmit = async (values: CreateRoleModel) => {
    try {
      if (sheetMode === "create") {
        await createRoleMutation.mutateAsync(values)
      } else if (selectedRole) {
        await updateRoleMutation.mutateAsync({
          roleId: selectedRole.id,
          payload: values,
        })
      }

      setIsSheetOpen(false)
      setSelectedRole(null)
    } catch {
      // Global mutation toast handles server-side failures.
    }
  }

  return (
    <>
      <PageHeaderCard
        icon={<RolesIcon />}
        title={t("identity.roles.page.title")}
        description={t("identity.roles.page.description")}
        titleMeta={totalItems}
        headerRight={
          <Button type="button" size="lg" onClick={handleOpenCreate}>
            <CreateIcon data-icon="inline-start" />
            {t("identity.roles.actions.newRole")}
          </Button>
        }
      />

      <SectionCard
        headerRight={
          selectedRoleIds.length > 0 ? (
            <DeleteConfirmDialog
              open={isDeleteDialogOpen}
              onOpenChange={setIsDeleteDialogOpen}
              title={t("identity.roles.deleteDialog.title")}
              description={t("identity.roles.deleteDialog.description", {
                count: selectedRoleIds.length,
              })}
              confirmLabel={t("identity.roles.deleteDialog.confirm")}
              cancelLabel={t("identity.roles.deleteDialog.cancel")}
              onConfirm={async () => {
                await handleDeleteSelected()
                setIsDeleteDialogOpen(false)
              }}
              trigger={
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  disabled={deleteRolesMutation.isPending}
                >
                  <DeleteIcon data-icon="inline-start" />
                  {t("identity.roles.actions.deleteSelected", {
                    count: selectedRoleIds.length,
                  })}
                </Button>
              }
            />
          ) : null
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
                  placeholder={t("identity.roles.searchPlaceholder")}
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
                          ? t("identity.roles.table.name")
                          : column.id === "description"
                            ? t("identity.roles.table.description")
                            : t("identity.roles.table.updatedAt")}
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
                    {t("identity.roles.filters.sortBy")}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <Button
                      type="button"
                      size="sm"
                      variant={
                        sorting[0]?.id === "name" ? "secondary" : "outline"
                      }
                      onClick={() => setSorting([{ id: "name", desc: false }])}
                    >
                      {t("identity.roles.table.name")}
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant={
                        sorting[0]?.id === "updated_at"
                          ? "secondary"
                          : "outline"
                      }
                      onClick={() =>
                        setSorting([{ id: "updated_at", desc: true }])
                      }
                    >
                      {t("identity.roles.table.updatedAt")}
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
                ? t("identity.roles.errors.loadRoles")
                : t("identity.roles.emptyRoles")
            }
            skeletonRows={5}
          />

          <DataTablePagination
            table={table}
            rowCount={totalItems}
            isFetching={isFetching}
            pageSizeOptions={pageSizeOptions}
            selectedCount={selectedRoleIds.length}
          />
        </div>
      </SectionCard>

      <RoleFormSheet
        open={isSheetOpen}
        onOpenChange={(open) => {
          setIsSheetOpen(open)
          if (!open) {
            setSelectedRole(null)
          }
        }}
        mode={sheetMode}
        role={selectedRole}
        pending={createRoleMutation.isPending || updateRoleMutation.isPending}
        schema={sheetMode === "create" ? createRoleSchema : updateRoleSchema}
        onSubmit={handleSubmit}
      />
    </>
  )
}
