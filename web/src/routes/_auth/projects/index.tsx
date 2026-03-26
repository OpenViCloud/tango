import { createFileRoute } from "@tanstack/react-router"
import { ProjectsPage } from "@/pages/auth/projects/projects-page"

export const Route = createFileRoute("/_auth/projects/")({
  component: ProjectsPage,
})
