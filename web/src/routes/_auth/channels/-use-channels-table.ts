import {
  type PaginationState,
  type RowSelectionState,
  type SortingState,
  type VisibilityState,
} from "@tanstack/react-table"
import { useDeferredValue, useMemo, useState } from "react"

import { useDataTable } from "@/components/data-table/use-data-table"
import {
  useDeleteChannels,
  useGetChannelList,
  useRestartChannel,
  useStartChannel,
  useStopChannel,
} from "@/hooks/api/use-channel"
import { getChannelColumns } from "@/routes/_auth/channels/-columns"
import { useTranslation } from "react-i18next"

const pageSizeOptions = [10, 20, 50] as const

export function useChannelsTable(query: string) {
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
  const deleteChannelsMutation = useDeleteChannels()
  const startChannelMutation = useStartChannel()
  const stopChannelMutation = useStopChannel()
  const restartChannelMutation = useRestartChannel()

  const orderBy = sorting[0]?.id === "kind" ? "kind" : "name"
  const ascending = sorting[0] ? !sorting[0].desc : true

  const { data, isLoading, isError, isFetching, refetch } = useGetChannelList({
    pageIndex: pagination.pageIndex,
    pageSize: pagination.pageSize,
    searchText: deferredQuery || undefined,
    orderBy,
    ascending,
  })

  const channels = useMemo(() => data?.items ?? [], [data?.items])
  const totalItems = data?.totalItems ?? 0
  const currentChannelIds = useMemo(
    () => new Set(channels.map((channel) => channel.id)),
    [channels]
  )
  const visibleRowSelection = useMemo(
    () =>
      Object.fromEntries(
        Object.entries(rowSelection).filter(
          ([id, selected]) => selected && currentChannelIds.has(id)
        )
      ),
    [currentChannelIds, rowSelection]
  )

  const isRuntimeActionPending =
    startChannelMutation.isPending ||
    stopChannelMutation.isPending ||
    restartChannelMutation.isPending

  const columns = useMemo(
    () =>
      getChannelColumns({
        t,
        sorting,
        actionPending: isRuntimeActionPending,
        onStart: (channelId: string) => {
          void startChannelMutation.mutateAsync(channelId)
        },
        onStop: (channelId: string) => {
          void stopChannelMutation.mutateAsync(channelId)
        },
        onRestart: (channelId: string) => {
          void restartChannelMutation.mutateAsync(channelId)
        },
      }),
    [
      isRuntimeActionPending,
      restartChannelMutation,
      sorting,
      startChannelMutation,
      stopChannelMutation,
      t,
    ]
  )

  const table = useDataTable({
    data: channels,
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

  const selectedChannelIds = Object.keys(visibleRowSelection).filter(
    (id) => visibleRowSelection[id]
  )

  const handleDeleteSelected = async () => {
    if (selectedChannelIds.length === 0) {
      return
    }

    try {
      await deleteChannelsMutation.mutateAsync(selectedChannelIds)
      setRowSelection({})
    } catch {
      // Global mutation toast handles server-side failures.
    }
  }

  return {
    table,
    totalItems,
    isLoading,
    isError,
    isFetching,
    refetch,
    selectedChannelIds,
    handleDeleteSelected,
    deleteChannelsMutation,
    startChannelMutation,
    stopChannelMutation,
    restartChannelMutation,
    pageSizeOptions,
    sorting,
    setPagination,
    setSorting,
  }
}
