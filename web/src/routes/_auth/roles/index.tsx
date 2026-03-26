import { createFileRoute } from "@tanstack/react-router"

import RolesPage from "@/pages/auth/roles/roles-page"

export const Route = createFileRoute("/_auth/roles/")({
  component: RolesPage,
})
