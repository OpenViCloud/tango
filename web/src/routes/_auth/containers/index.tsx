import { createFileRoute } from "@tanstack/react-router"
import { ContainersPage } from "@/pages/auth/containers/containers-page"

export const Route = createFileRoute("/_auth/containers/")({
  component: ContainersPage,
})
