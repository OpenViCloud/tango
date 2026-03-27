import { createFileRoute } from "@tanstack/react-router"
import { z } from "zod"

import { ResourceCreationPage } from "@/pages/auth/projects/resource-creation-page"

const searchSchema = z.object({
  projectId: z.string().optional(),
})

export const Route = createFileRoute(
  "/_auth/environments/$envId/resources/new"
)({
  validateSearch: searchSchema,
  component: ResourceCreationRoute,
})

function ResourceCreationRoute() {
  const { envId } = Route.useParams()
  const { projectId } = Route.useSearch()
  return <ResourceCreationPage envId={envId} projectId={projectId ?? ""} />
}
