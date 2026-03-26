import { createFileRoute } from "@tanstack/react-router"

import DashboardPage from "@/pages/auth/dashboard-page"

export const Route = createFileRoute("/_auth/dashboard")({
  component: DashboardPage,
})
