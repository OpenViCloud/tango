import { useEffect, useRef, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import type { BootstrapClusterModel, ClusterNodeModel } from "@/@types/models/server"
import { getErrorMessage } from "@/lib/get-error-message"
import { clusterService } from "@/services/api/cluster-service"

export const CLUSTER_QUERY_KEYS = {
  list: () => ["clusters"],
  detail: (id: string) => ["clusters", id],
}

export const useGetClusterList = () =>
  useQuery({
    queryKey: CLUSTER_QUERY_KEYS.list(),
    queryFn: () => clusterService.list(),
  })

export const useGetCluster = (id: string) =>
  useQuery({
    queryKey: CLUSTER_QUERY_KEYS.detail(id),
    queryFn: () => clusterService.getById(id),
    enabled: Boolean(id),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === "provisioning" || status === "pending" ? 3000 : false
    },
  })

export const useBootstrapCluster = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: BootstrapClusterModel) =>
      clusterService.bootstrap(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: CLUSTER_QUERY_KEYS.list(),
      })
    },
    onError: (error) => {
      toast.error(getErrorMessage(error))
    },
  })
}

export const useInventoryPreview = () =>
  useMutation({
    mutationFn: ({
      clusterId,
      nodes,
    }: {
      clusterId: string
      nodes: ClusterNodeModel[]
    }) => clusterService.inventoryPreview(clusterId, nodes),
  })

export const useDeleteCluster = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, purge }: { id: string; purge?: boolean }) =>
      clusterService.delete(id, purge),
    onSuccess: (_data, { purge }) => {
      void queryClient.invalidateQueries({
        queryKey: CLUSTER_QUERY_KEYS.list(),
      })
      toast.success(purge ? "Cluster removed and K8s uninstall started on nodes." : "Cluster removed.")
    },
    onError: (error) => {
      toast.error(getErrorMessage(error))
    },
  })
}

// ── WebSocket log streaming ──────────────────────────────────────────────────

type WsMsg =
  | { t: "log"; d: string }
  | { t: "done"; status?: string }
  | { t: "error"; d: string }

export type ClusterLogState = {
  lines: string[]
  done: boolean
  connected: boolean
}

export function useClusterLogs(clusterId: string | null): ClusterLogState {
  const [lines, setLines] = useState<string[]>([])
  const [done, setDone] = useState(false)
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    if (!clusterId) return

    setLines([])
    setDone(false)

    const proto = window.location.protocol === "https:" ? "wss" : "ws"
    const url = `${proto}://${window.location.host}/api/ws/clusters/${clusterId}/logs`
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => setConnected(true)

    ws.onmessage = (e) => {
      const msg: WsMsg = JSON.parse(e.data as string)
      if (msg.t === "log") {
        setLines((prev) => [...prev, msg.d ?? ""])
      } else if (msg.t === "done") {
        setDone(true)
        setConnected(false)
      } else if (msg.t === "error") {
        setLines((prev) => [...prev, `[ERROR] ${msg.d}`])
      }
    }

    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false)

    return () => ws.close()
  }, [clusterId])

  return { lines, done, connected }
}
