import { createFileRoute } from "@tanstack/react-router"
import { BootstrapClusterPage } from "@/pages/auth/clusters/bootstrap-cluster-page"

export const Route = createFileRoute("/_auth/clusters/new")({
  component: BootstrapClusterPage,
})
