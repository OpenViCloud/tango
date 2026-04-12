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

// ── Kubernetes resource queries ──────────────────────────────────────────────

export const KUBE_QUERY_KEYS = {
  namespaces: (id: string) => ["clusters", id, "namespaces"],
  pods: (id: string, namespace?: string) => ["clusters", id, "pods", namespace ?? ""],
  services: (id: string, namespace?: string) => ["clusters", id, "services", namespace ?? ""],
  volumes: (id: string) => ["clusters", id, "volumes"],
  volumeClaims: (id: string, namespace?: string) => ["clusters", id, "volume-claims", namespace ?? ""],
}

export const useKubeNamespaces = (clusterId: string, enabled = true) =>
  useQuery({
    queryKey: KUBE_QUERY_KEYS.namespaces(clusterId),
    queryFn: () => clusterService.listNamespaces(clusterId),
    enabled: Boolean(clusterId) && enabled,
  })

export const useKubePods = (clusterId: string, namespace?: string, enabled = true) =>
  useQuery({
    queryKey: KUBE_QUERY_KEYS.pods(clusterId, namespace),
    queryFn: () => clusterService.listPods(clusterId, namespace),
    enabled: Boolean(clusterId) && enabled,
  })

export const useKubeServices = (clusterId: string, namespace?: string, enabled = true) =>
  useQuery({
    queryKey: KUBE_QUERY_KEYS.services(clusterId, namespace),
    queryFn: () => clusterService.listServices(clusterId, namespace),
    enabled: Boolean(clusterId) && enabled,
  })

export const useKubePersistentVolumes = (clusterId: string, enabled = true) =>
  useQuery({
    queryKey: KUBE_QUERY_KEYS.volumes(clusterId),
    queryFn: () => clusterService.listPersistentVolumes(clusterId),
    enabled: Boolean(clusterId) && enabled,
  })

export const useKubePersistentVolumeClaims = (clusterId: string, namespace?: string, enabled = true) =>
  useQuery({
    queryKey: KUBE_QUERY_KEYS.volumeClaims(clusterId, namespace),
    queryFn: () => clusterService.listPersistentVolumeClaims(clusterId, namespace),
    enabled: Boolean(clusterId) && enabled,
  })

export const useCreateKubePod = (clusterId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      namespace,
      name,
      image,
      labels,
    }: {
      namespace: string
      name: string
      image: string
      labels?: Record<string, string>
    }) => clusterService.createPod(clusterId, namespace, { name, image, labels }),
    onSuccess: (_data, { namespace }) => {
      void queryClient.invalidateQueries({
        queryKey: KUBE_QUERY_KEYS.pods(clusterId, namespace),
      })
      void queryClient.invalidateQueries({
        queryKey: KUBE_QUERY_KEYS.pods(clusterId),
      })
      toast.success("Pod created.")
    },
    onError: (error) => {
      toast.error(getErrorMessage(error))
    },
  })
}

export const useCreateKubeService = (clusterId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      namespace,
      name,
      type,
      selector,
      ports,
    }: {
      namespace: string
      name: string
      type: string
      selector: Record<string, string>
      ports: Array<{ name: string; port: number; target_port: string; protocol: string }>
    }) => clusterService.createService(clusterId, namespace, { name, type, selector, ports }),
    onSuccess: (_data, { namespace }) => {
      void queryClient.invalidateQueries({
        queryKey: KUBE_QUERY_KEYS.services(clusterId, namespace),
      })
      void queryClient.invalidateQueries({
        queryKey: KUBE_QUERY_KEYS.services(clusterId),
      })
      toast.success("Service created.")
    },
    onError: (error) => {
      toast.error(getErrorMessage(error))
    },
  })
}

export const useDeleteKubePod = (clusterId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ namespace, name }: { namespace: string; name: string }) =>
      clusterService.deletePod(clusterId, namespace, name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["clusters", clusterId, "pods"] })
      toast.success("Pod deleted.")
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  })
}

export const useDeleteKubeService = (clusterId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ namespace, name }: { namespace: string; name: string }) =>
      clusterService.deleteService(clusterId, namespace, name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["clusters", clusterId, "services"] })
      toast.success("Service deleted.")
    },
    onError: (error) => toast.error(getErrorMessage(error)),
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
