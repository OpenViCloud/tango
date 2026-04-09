import { useState } from "react"
import { useNavigate } from "@tanstack/react-router"
import { NetworkIcon, PlusIcon, Trash2Icon, DownloadIcon, AlertTriangleIcon } from "lucide-react"

import type { ClusterModel } from "@/@types/models/server"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { useGetClusterList, useDeleteCluster } from "@/hooks/api/use-cluster"
import { clusterService } from "@/services/api/cluster-service"
import { toast } from "sonner"

export function ClustersPage() {
  const { data: clusters, isLoading } = useGetClusterList()
  const navigate = useNavigate()

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<NetworkIcon className="size-6" />}
        title="Bootstrap Cluster"
        description="Provision and manage Kubernetes clusters."
        headerRight={
          <Button size="sm" onClick={() => void navigate({ to: "/clusters/new" })}>
            <PlusIcon className="size-4" />
            Bootstrap Cluster
          </Button>
        }
      />

      <SectionCard
        icon={<NetworkIcon className="size-5" />}
        title="Clusters"
        description="All provisioned K8s clusters."
      >
        {isLoading ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : !clusters?.length ? (
          <div className="flex flex-col items-center gap-3 py-8">
            <p className="text-sm text-muted-foreground">No clusters yet.</p>
            <Button
              size="sm"
              onClick={() => void navigate({ to: "/clusters/new" })}
            >
              <PlusIcon className="size-4" />
              Bootstrap your first cluster
            </Button>
          </div>
        ) : (
          <ClusterTable clusters={clusters} />
        )}
      </SectionCard>
    </div>
  )
}

function ClusterTable({ clusters }: { clusters: ClusterModel[] }) {
  const { mutate: remove, isPending } = useDeleteCluster()
  const navigate = useNavigate()
  const [deleteTarget, setDeleteTarget] = useState<ClusterModel | null>(null)

  const downloadKubeconfig = async (cluster: ClusterModel) => {
    try {
      const blob = await clusterService.downloadKubeconfig(cluster.id)
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = `kubeconfig-${cluster.name}.yaml`
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      toast.error("Kubeconfig not available yet.")
    }
  }

  const handleDelete = (purge: boolean) => {
    if (!deleteTarget) return
    remove(
      { id: deleteTarget.id, purge },
      { onSettled: () => setDeleteTarget(null) },
    )
  }

  return (
    <>
      <div className="overflow-x-auto rounded-xl border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/40 text-left text-xs tracking-wide text-muted-foreground uppercase">
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">K8s Version</th>
              <th className="px-4 py-3">Nodes</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {clusters.map((cluster) => (
              <tr
                key={cluster.id}
                className="cursor-pointer hover:bg-muted/20"
                onClick={() =>
                  void navigate({
                    to: "/clusters/$clusterId",
                    params: { clusterId: cluster.id },
                  })
                }
              >
                <td className="px-4 py-3 font-medium">{cluster.name}</td>
                <td className="px-4 py-3 font-mono text-xs">{cluster.k8s_version}</td>
                <td className="px-4 py-3 text-muted-foreground">{cluster.nodes.length}</td>
                <td className="px-4 py-3">
                  <ClusterStatusBadge status={cluster.status} />
                </td>
                <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
                  <div className="flex items-center gap-2">
                    {cluster.status === "ready" && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => void downloadKubeconfig(cluster)}
                      >
                        <DownloadIcon className="size-3" />
                        kubeconfig
                      </Button>
                    )}
                    <Button
                      size="icon"
                      variant="ghost"
                      className="text-destructive hover:text-destructive"
                      onClick={() => setDeleteTarget(cluster)}
                      title="Delete cluster"
                    >
                      <Trash2Icon className="size-4" />
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <DeleteClusterDialog
        cluster={deleteTarget}
        isPending={isPending}
        onDelete={handleDelete}
        onClose={() => setDeleteTarget(null)}
      />
    </>
  )
}

function DeleteClusterDialog({
  cluster,
  isPending,
  onDelete,
  onClose,
}: {
  cluster: ClusterModel | null
  isPending: boolean
  onDelete: (purge: boolean) => void
  onClose: () => void
}) {
  return (
    <Dialog open={Boolean(cluster)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AlertTriangleIcon className="size-5 text-destructive" />
            Delete cluster &quot;{cluster?.name}&quot;
          </DialogTitle>
          <DialogDescription>
            Choose how to delete this cluster. This action cannot be undone.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3 py-2">
          <div className="rounded-lg border p-4">
            <p className="text-sm font-medium">Delete only</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Remove the cluster record from Tango. Kubernetes remains installed on the VPS nodes.
            </p>
          </div>
          <div className="rounded-lg border border-destructive/40 bg-destructive/5 p-4">
            <p className="text-sm font-medium text-destructive">Purge (recommended)</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Remove the cluster record <strong>and</strong> uninstall Kubernetes from all VPS nodes in the background.
            </p>
          </div>
        </div>

        <DialogFooter className="flex gap-2 sm:justify-between">
          <Button variant="outline" onClick={onClose} disabled={isPending}>
            Cancel
          </Button>
          <div className="flex gap-2">
            <Button
              variant="ghost"
              className="text-destructive hover:text-destructive"
              onClick={() => onDelete(false)}
              disabled={isPending}
            >
              Delete only
            </Button>
            <Button
              variant="destructive"
              onClick={() => onDelete(true)}
              disabled={isPending}
            >
              Purge
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function ClusterStatusBadge({ status }: { status: string }) {
  switch (status) {
    case "ready":
      return <Badge variant="default">Ready</Badge>
    case "provisioning":
      return <Badge variant="secondary" className="animate-pulse">Provisioning</Badge>
    case "error":
      return <Badge variant="destructive">Error</Badge>
    default:
      return <Badge variant="secondary">Pending</Badge>
  }
}
