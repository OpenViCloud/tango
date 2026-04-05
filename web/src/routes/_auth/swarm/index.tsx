import { createFileRoute } from "@tanstack/react-router"

import { SwarmNodesPage } from "@/pages/auth/swarm/swarm-nodes-page"

export const Route = createFileRoute("/_auth/swarm/")({
  component: SwarmNodesPage,
})
