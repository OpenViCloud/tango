import { useEffect, useRef } from "react"
import { useNavigate } from "@tanstack/react-router"
import { NetworkIcon, ArrowLeftIcon, DownloadIcon, ServerIcon, FileTextIcon } from "lucide-react"
import { useQuery } from "@tanstack/react-query"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { useGetCluster, useClusterLogs } from "@/hooks/api/use-cluster"
import { useGetServerList } from "@/hooks/api/use-server"
import { clusterService } from "@/services/api/cluster-service"
import { ClusterStatusBadge } from "./clusters-page"
import type { ClusterModel } from "@/@types/models/server"

interface ClusterDetailPageProps {
  clusterId: string
}

export function ClusterDetailPage({ clusterId }: ClusterDetailPageProps) {
  const navigate = useNavigate()
  const { data: cluster, isLoading } = useGetCluster(clusterId)
  const { data: servers } = useGetServerList()
  const isProvisioning = cluster?.status === "provisioning" || cluster?.status === "pending"
  const { lines, done, connected } = useClusterLogs(isProvisioning ? clusterId : null)

  const serverMap = Object.fromEntries((servers ?? []).map((s) => [s.id, s]))

  const downloadKubeconfig = async () => {
    try {
      const blob = await clusterService.downloadKubeconfig(clusterId)
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = `kubeconfig-${cluster?.name ?? clusterId}.yaml`
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      toast.error("Kubeconfig not available yet.")
    }
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    )
  }

  if (!cluster) {
    return <p className="text-sm text-muted-foreground">Cluster not found.</p>
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<NetworkIcon className="size-6" />}
        title={cluster.name}
        titleMeta={cluster.k8s_version}
        description={`Pod CIDR: ${cluster.pod_cidr}`}
        headerRight={
          <div className="flex gap-2">
            {cluster.status === "ready" && (
              <Button size="sm" variant="outline" onClick={() => void downloadKubeconfig()}>
                <DownloadIcon className="size-4" />
                kubeconfig
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={() => void navigate({ to: "/clusters" })}
            >
              <ArrowLeftIcon className="size-4" />
              Back
            </Button>
          </div>
        }
      />

      {/* Status */}
      <SectionCard title="Status" icon={<NetworkIcon className="size-5" />}>
        <div className="flex items-center gap-3">
          <ClusterStatusBadge status={cluster.status} />
          {cluster.error_msg && (
            <p className="text-sm text-destructive">{cluster.error_msg}</p>
          )}
        </div>
      </SectionCard>

      {/* Nodes */}
      <SectionCard title="Nodes" icon={<ServerIcon className="size-5" />}>
        <div className="overflow-x-auto rounded-xl border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left text-xs tracking-wide text-muted-foreground uppercase">
                <th className="px-4 py-3">Server</th>
                <th className="px-4 py-3">IP</th>
                <th className="px-4 py-3">Role</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {cluster.nodes.map((node) => {
                const server = serverMap[node.server_id]
                return (
                  <tr key={node.server_id} className="hover:bg-muted/20">
                    <td className="px-4 py-3 font-medium">
                      {server?.name ?? node.server_id}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                      {server?.public_ip ?? "—"}
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant={node.role === "master" ? "default" : "secondary"} className="capitalize">
                        {node.role}
                      </Badge>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </SectionCard>

      {/* Inventory */}
      <InventorySection cluster={cluster} />

      {/* Live logs */}
      {(isProvisioning || lines.length > 0) && (
        <SectionCard
          title="Provisioning Logs"
          icon={<NetworkIcon className="size-5" />}
          description={connected ? "Live streaming..." : done ? "Provisioning complete." : ""}
          headerRight={
            connected ? (
              <Badge variant="secondary" className="animate-pulse">Live</Badge>
            ) : null
          }
        >
          <LogViewer lines={lines} />
        </SectionCard>
      )}
    </div>
  )
}

function InventorySection({ cluster }: { cluster: ClusterModel }) {
  const { data, isLoading } = useQuery({
    queryKey: ["clusters", cluster.id, "inventory"],
    queryFn: () => clusterService.inventoryPreview(cluster.id, cluster.nodes),
    staleTime: Infinity,
  })

  return (
    <SectionCard
      title="Ansible Inventory"
      icon={<FileTextIcon className="size-5" />}
      description="inventory.ini used for provisioning"
    >
      {isLoading ? (
        <Skeleton className="h-32 w-full" />
      ) : (
        <pre className="overflow-x-auto rounded-lg border bg-zinc-950 p-4 font-mono text-xs text-zinc-100 whitespace-pre">
          {data?.inventory ?? "—"}
        </pre>
      )}
    </SectionCard>
  )
}

function LogViewer({ lines }: { lines: string[] }) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [lines])

  if (!lines.length) {
    return (
      <div className="flex h-32 items-center justify-center rounded-lg border bg-muted/20">
        <p className="text-xs text-muted-foreground animate-pulse">Waiting for logs...</p>
      </div>
    )
  }

  return (
    <div className="max-h-[500px] overflow-y-auto rounded-lg border bg-zinc-950 p-4 font-mono text-xs text-zinc-100">
      {lines.map((line, i) => (
        <div key={i} className={!line ? "" : line.startsWith("[ERROR]") ? "text-red-400" : line.startsWith(">>>") ? "text-yellow-300 font-semibold mt-2" : line.startsWith("===") ? "text-green-400 font-semibold" : ""}>
          {line}
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  )
}
