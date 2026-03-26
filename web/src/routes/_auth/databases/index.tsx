import { createFileRoute } from "@tanstack/react-router"
import { DatabasesPage } from "@/pages/auth/databases/databases-page"

export const Route = createFileRoute("/_auth/databases/")({
  component: DatabasesPage,
})
