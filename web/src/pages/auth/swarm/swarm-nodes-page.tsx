import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { useSwarmNodes, useSwarmStatus } from "@/hooks/api/use-swarm"
import type { SwarmNodeModel } from "@/@types/models/swarm"
import { NetworkIcon } from "lucide-react"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"

export function SwarmNodesPage() {
  const { data: status, isLoading: statusLoading } = useSwarmStatus()
  const isManager = status?.is_manager ?? false
  const { data: nodes, isLoading: nodesLoading } = useSwarmNodes(isManager)

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<NetworkIcon className="size-6" />}
        title="Swarm Cluster"
        description="View Docker Swarm nodes and cluster status."
      />

      <SectionCard
        icon={<NetworkIcon className="size-5" />}
        title="Cluster status"
        description="Whether this node is an active swarm manager."
      >
        {statusLoading ? (
          <Skeleton className="h-6 w-32" />
        ) : (
          <div className="flex items-center gap-3">
            <Badge variant={isManager ? "default" : "secondary"}>
              {isManager ? "Manager" : "Not a manager"}
            </Badge>
            {!isManager && (
              <p className="text-sm text-muted-foreground">
                This node is not an active swarm manager. Join or initialise a
                swarm to manage cluster resources.
              </p>
            )}
          </div>
        )}
      </SectionCard>

      {isManager && (
        <SectionCard
          icon={<NetworkIcon className="size-5" />}
          title="Nodes"
          description="All nodes registered in the cluster."
        >
          <NodeTable nodes={nodes ?? []} isLoading={nodesLoading} />
        </SectionCard>
      )}
    </div>
  )
}

function NodeTable({
  nodes,
  isLoading,
}: {
  nodes: SwarmNodeModel[]
  isLoading: boolean
}) {
  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-4/5" />
      </div>
    )
  }

  if (nodes.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No nodes found in cluster.
      </p>
    )
  }

  return (
    <div className="overflow-x-auto rounded-xl border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs tracking-wide text-muted-foreground uppercase">
            <th className="px-4 py-3">Hostname</th>
            <th className="px-4 py-3">Role</th>
            <th className="px-4 py-3">State</th>
            <th className="px-4 py-3">Availability</th>
            <th className="px-4 py-3">Manager addr</th>
            <th className="px-4 py-3 font-mono text-xs">ID</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {nodes.map((node) => (
            <tr key={node.id} className="hover:bg-muted/20">
              <td className="px-4 py-3 font-medium">{node.hostname}</td>
              <td className="px-4 py-3">
                <Badge
                  variant={node.role === "manager" ? "default" : "secondary"}
                >
                  {node.role}
                </Badge>
              </td>
              <td className="px-4 py-3">
                <NodeStateBadge state={node.state} />
              </td>
              <td className="px-4 py-3">
                <Badge
                  variant={
                    node.availability === "active" ? "outline" : "secondary"
                  }
                >
                  {node.availability}
                </Badge>
              </td>
              <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                {node.manager_addr ?? "—"}
              </td>
              <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                {node.id.slice(0, 12)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function NodeStateBadge({ state }: { state: string }) {
  const variant =
    state === "ready" ? "default" : state === "down" ? "warning" : "secondary"
  return <Badge variant={variant}>{state}</Badge>
}
