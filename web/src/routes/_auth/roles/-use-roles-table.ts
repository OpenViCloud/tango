import {
  type PaginationState,
  type RowSelectionState,
  type SortingState,
  type VisibilityState,
} from "@tanstack/react-table"
import { useDeferredValue, useMemo, useState } from "react"

import type { RoleModel } from "@/@types/models"
import { useDataTable } from "@/components/data-table/use-data-table"
import { useDeleteRoles, useGetRoleList } from "@/hooks/api/use-role"
import { getRoleColumns } from "@/routes/_auth/roles/-columns"
import { useTranslation } from "react-i18next"

const pageSizeOptions = [10, 20, 50] as const

export function useRolesTable(
  query: string,
  onEdit: (role: RoleModel) => void
) {
  const { t } = useTranslation()
  const deferredQuery = useDeferredValue(query.trim())
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [sorting, setSorting] = useState<SortingState>([
    { id: "name", desc: false },
  ])
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const deleteRolesMutation = useDeleteRoles()

  const orderBy = sorting[0]?.id === "updated_at" ? "updated_at" : "name"
  const ascending = sorting[0] ? !sorting[0].desc : true

  const { data, isLoading, isError, isFetching, refetch } = useGetRoleList({
    pageIndex: pagination.pageIndex,
    pageSize: pagination.pageSize,
    searchText: deferredQuery || undefined,
    orderBy,
    ascending,
  })

  const roles = useMemo(() => data?.items ?? [], [data?.items])
  const totalItems = data?.totalItems ?? 0
  const currentRoleIds = useMemo(() => new Set(roles.map((role) => role.id)), [roles])
  const visibleRowSelection = useMemo(
    () =>
      Object.fromEntries(
        Object.entries(rowSelection).filter(
          ([id, selected]) => selected && currentRoleIds.has(id)
        )
      ),
    [currentRoleIds, rowSelection]
  )

  const columns = useMemo(
    () => getRoleColumns({ t, sorting, onEdit }),
    [onEdit, sorting, t]
  )

  const table = useDataTable({
    data: roles,
    columns,
    rowCount: totalItems,
    getRowId: (row) => row.id,
    manualPagination: true,
    manualSorting: true,
    enableRowSelection: (row) => !row.original.is_system,
    onPaginationChange: setPagination,
    onSortingChange: setSorting,
    onRowSelectionChange: setRowSelection,
    onColumnVisibilityChange: setColumnVisibility,
    state: {
      pagination,
      sorting,
      rowSelection: visibleRowSelection,
      columnVisibility,
    },
  })

  const selectedRoleIds = Object.keys(visibleRowSelection).filter(
    (id) => visibleRowSelection[id]
  )

  const handleDeleteSelected = async () => {
    if (selectedRoleIds.length === 0) {
      return
    }

    try {
      await deleteRolesMutation.mutateAsync(selectedRoleIds)
      setRowSelection({})
    } catch {
      // Mutation handles its own toast.
    }
  }

  return {
    table,
    totalItems,
    roles,
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
  }
}
