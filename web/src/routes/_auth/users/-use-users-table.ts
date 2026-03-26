import {
  type PaginationState,
  type RowSelectionState,
  type SortingState,
  type VisibilityState,
} from "@tanstack/react-table"
import { useDeferredValue, useMemo, useState } from "react"

import { useDataTable } from "@/components/data-table/use-data-table"
import { useDeleteUsers, useGetUserList } from "@/hooks/api/use-user"
import { getUserColumns } from "@/routes/_auth/users/-columns"
import { useTranslation } from "react-i18next"

const pageSizeOptions = [10, 20, 50] as const

export function useUsersTable(query: string) {
  const { t } = useTranslation()
  const deferredQuery = useDeferredValue(query.trim())
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [sorting, setSorting] = useState<SortingState>([
    { id: "email", desc: false },
  ])
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const deleteUsersMutation = useDeleteUsers()

  const orderBy = sorting[0]?.id === "first_name" ? "first_name" : "email"
  const ascending = sorting[0] ? !sorting[0].desc : true

  const { data, isLoading, isError, isFetching, refetch } = useGetUserList({
    pageIndex: pagination.pageIndex,
    pageSize: pagination.pageSize,
    searchText: deferredQuery || undefined,
    orderBy,
    ascending,
  })

  const users = useMemo(() => data?.items ?? [], [data?.items])
  const totalItems = data?.totalItems ?? 0
  const currentUserIds = useMemo(() => new Set(users.map((user) => user.id)), [users])
  const visibleRowSelection = useMemo(
    () =>
      Object.fromEntries(
        Object.entries(rowSelection).filter(([id, selected]) => selected && currentUserIds.has(id))
      ),
    [currentUserIds, rowSelection]
  )

  const columns = useMemo(() => getUserColumns({ t, sorting }), [sorting, t])

  const table = useDataTable({
    data: users,
    columns,
    rowCount: totalItems,
    getRowId: (row) => row.id,
    manualPagination: true,
    manualSorting: true,
    enableRowSelection: true,
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

  const selectedUserIds = Object.keys(visibleRowSelection).filter(
    (id) => visibleRowSelection[id]
  )
  const activeCount = users.filter((user) => user.status === "active").length
  const bannedCount = users.filter((user) => user.status === "banned").length

  const handleDeleteSelected = async () => {
    if (selectedUserIds.length === 0) {
      return
    }

    try {
      await deleteUsersMutation.mutateAsync(selectedUserIds)
      setRowSelection({})
    } catch {
      // Mutation handles its own toast.
    }
  }

  return {
    table,
    totalItems,
    activeCount,
    bannedCount,
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
  }
}
