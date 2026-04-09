import { createFileRoute } from "@tanstack/react-router"
import { ClusterDetailPage } from "@/pages/auth/clusters/cluster-detail-page"

export const Route = createFileRoute("/_auth/clusters/$clusterId")({
  component: function ClusterDetailRoute() {
    const { clusterId } = Route.useParams()
    return <ClusterDetailPage clusterId={clusterId} />
  },
})
