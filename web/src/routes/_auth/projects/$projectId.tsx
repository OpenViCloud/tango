import { createFileRoute } from "@tanstack/react-router"
import { ProjectDetailPage } from "@/pages/auth/projects/project-detail-page"

export const Route = createFileRoute("/_auth/projects/$projectId")({
  component: ProjectDetailPage,
})
