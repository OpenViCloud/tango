import { useState } from "react"
import { useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { NetworkIcon, ArrowLeftIcon, EyeIcon } from "lucide-react"

import type { ServerModel, ClusterNodeModel } from "@/@types/models/server"

const clusterConfigSchema = z.object({
  name: z.string().min(1, "Name is required"),
  k8s_version: z.string().optional(),
  pod_cidr: z.string().optional(),
})
type ClusterConfigForm = z.infer<typeof clusterConfigSchema>
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { useGetServerList } from "@/hooks/api/use-server"
import { useBootstrapCluster } from "@/hooks/api/use-cluster"
import { clusterService } from "@/services/api/cluster-service"

export function BootstrapClusterPage() {
  const navigate = useNavigate()
  const { data: servers, isLoading: serversLoading } = useGetServerList()
  const { mutate: bootstrap, isPending } = useBootstrapCluster()

  const [nodeRoles, setNodeRoles] = useState<Record<string, ClusterNodeModel["role"] | "none">>({})
  const [inventoryPreview, setInventoryPreview] = useState<string | null>(null)
  const [showPreview, setShowPreview] = useState(false)
  const [nodesError, setNodesError] = useState<string | null>(null)

  const form = useForm<ClusterConfigForm>({
    resolver: zodResolver(clusterConfigSchema),
    defaultValues: {
      k8s_version: "v1.30",
      pod_cidr: "192.168.0.0/16",
    },
  })

  const connectedServers = servers?.filter((s) => s.status === "connected") ?? []

  const toggleRole = (serverId: string, role: ClusterNodeModel["role"] | "none") => {
    setNodeRoles((prev) => ({ ...prev, [serverId]: role }))
    setNodesError(null)
  }

  const buildNodes = (): ClusterNodeModel[] =>
    Object.entries(nodeRoles)
      .filter(([, role]) => role !== "none")
      .map(([server_id, role]) => ({ server_id, role: role as ClusterNodeModel["role"] }))

  const handlePreview = async () => {
    const nodes = buildNodes()
    if (!nodes.length) return
    // Use a temporary placeholder cluster id for preview
    try {
      const result = await clusterService.inventoryPreview("preview", nodes)
      setInventoryPreview(result.inventory)
      setShowPreview(true)
    } catch {
      // preview may fail without a real cluster id — show a static message
      setInventoryPreview("(Preview unavailable — create the cluster first)")
      setShowPreview(true)
    }
  }

  const onSubmit = (data: ClusterConfigForm) => {
    const nodes = buildNodes()
    if (!nodes.length) {
      setNodesError("Select at least one node role.")
      return
    }
    const masterCount = nodes.filter((n) => n.role === "master").length
    if (masterCount !== 1) {
      setNodesError("Exactly one master node is required.")
      return
    }
    setNodesError(null)

    bootstrap(
      { ...data, nodes },
      {
        onSuccess: (cluster) => {
          void navigate({ to: "/clusters/$clusterId", params: { clusterId: cluster.id } })
        },
      },
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<NetworkIcon className="size-6" />}
        title="Bootstrap Cluster"
        description="Provision a new Kubernetes cluster on your servers."
        headerRight={
          <Button variant="outline" size="sm" onClick={() => void navigate({ to: "/clusters" })}>
            <ArrowLeftIcon className="size-4" />
            Back
          </Button>
        }
      />

      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-6">
        {/* Cluster config */}
        <SectionCard title="Cluster Configuration" icon={<NetworkIcon className="size-5" />}>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div className="flex flex-col gap-1.5 sm:col-span-1">
              <Label htmlFor="name">Cluster Name</Label>
              <Input id="name" placeholder="production-k8s" {...form.register("name")} />
              {form.formState.errors.name && (
                <p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
              )}
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="k8s_version">K8s Version</Label>
              <Input id="k8s_version" placeholder="v1.30" {...form.register("k8s_version")} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="pod_cidr">Pod CIDR</Label>
              <Input id="pod_cidr" placeholder="192.168.0.0/16" {...form.register("pod_cidr")} />
            </div>
          </div>
        </SectionCard>

        {/* Node assignment */}
        <SectionCard
          title="Node Assignment"
          icon={<NetworkIcon className="size-5" />}
          description="Select connected servers and assign Master or Worker roles."
        >
          {serversLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
            </div>
          ) : !connectedServers.length ? (
            <div className="rounded-lg border border-dashed p-6 text-center">
              <p className="text-sm text-muted-foreground">
                No connected servers. Go to{" "}
                <button
                  type="button"
                  className="text-primary underline"
                  onClick={() => void navigate({ to: "/servers" })}
                >
                  Servers
                </button>{" "}
                and verify SSH connectivity first.
              </p>
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {connectedServers.map((server) => (
                <NodeRoleRow
                  key={server.id}
                  server={server}
                  role={nodeRoles[server.id] ?? "none"}
                  onRoleChange={(role) => toggleRole(server.id, role)}
                />
              ))}
              {nodesError && (
                <p className="text-xs text-destructive">{nodesError}</p>
              )}
            </div>
          )}
        </SectionCard>

        {/* Inventory preview */}
        <SectionCard title="Inventory Preview" icon={<EyeIcon className="size-5" />} description="Review the generated Ansible inventory before provisioning.">
          <div className="flex flex-col gap-3">
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="self-start"
              onClick={handlePreview}
              disabled={buildNodes().length === 0}
            >
              <EyeIcon className="size-4" />
              {showPreview ? "Refresh Preview" : "Show Preview"}
            </Button>
            {showPreview && inventoryPreview && (
              <pre className="overflow-x-auto rounded-lg border bg-muted/40 p-4 text-xs text-muted-foreground">
                {inventoryPreview}
              </pre>
            )}
          </div>
        </SectionCard>

        {/* Submit */}
        <div className="flex justify-end gap-3">
          <Button
            type="button"
            variant="outline"
            onClick={() => void navigate({ to: "/clusters" })}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={isPending || !connectedServers.length}>
            {isPending ? "Bootstrapping..." : "Bootstrap Cluster"}
          </Button>
        </div>
      </form>
    </div>
  )
}

function NodeRoleRow({
  server,
  role,
  onRoleChange,
}: {
  server: ServerModel
  role: ClusterNodeModel["role"] | "none"
  onRoleChange: (role: ClusterNodeModel["role"] | "none") => void
}) {
  return (
    <div className="flex items-center justify-between rounded-lg border px-4 py-3">
      <div className="flex flex-col gap-0.5">
        <span className="font-medium text-sm">{server.name}</span>
        <span className="font-mono text-xs text-muted-foreground">{server.public_ip}</span>
      </div>
      <div className="flex gap-2">
        {(["none", "master", "worker"] as const).map((r) => (
          <button
            key={r}
            type="button"
            onClick={() => onRoleChange(r)}
            className="focus:outline-none"
          >
            <Badge
              variant={role === r ? (r === "master" ? "default" : r === "worker" ? "secondary" : "outline") : "outline"}
              className={`cursor-pointer capitalize transition-opacity ${role !== r ? "opacity-40 hover:opacity-70" : ""}`}
            >
              {r === "none" ? "—" : r}
            </Badge>
          </button>
        ))}
      </div>
    </div>
  )
}
