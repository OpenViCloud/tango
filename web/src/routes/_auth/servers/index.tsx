import { createFileRoute } from "@tanstack/react-router"
import { ServersPage } from "@/pages/auth/servers/servers-page"

export const Route = createFileRoute("/_auth/servers/")({
  component: ServersPage,
})
