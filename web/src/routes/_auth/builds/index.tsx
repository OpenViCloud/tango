import { createFileRoute } from "@tanstack/react-router"
import BuildsPage from "@/pages/auth/builds/builds-page"

export const Route = createFileRoute("/_auth/builds/")({
  component: BuildsPage,
})
