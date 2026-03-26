import { createFileRoute } from "@tanstack/react-router"

import ResourceDetailPage from "@/pages/auth/projects/resource-detail-page"

export const Route = createFileRoute("/_auth/resources/$resourceId")({
  component: ResourceDetailRoute,
})

function ResourceDetailRoute() {
  const { resourceId } = Route.useParams()
  return <ResourceDetailPage resourceId={resourceId} />
}
