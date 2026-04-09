import { createFileRoute } from "@tanstack/react-router"
import { ClustersPage } from "@/pages/auth/clusters/clusters-page"

export const Route = createFileRoute("/_auth/clusters/")({
  component: ClustersPage,
})
