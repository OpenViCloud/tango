import { createFileRoute } from "@tanstack/react-router"

import { SourcesPage } from "@/pages/auth/sources/sources-page"

export const Route = createFileRoute("/_auth/sources/")({
  component: SourcesPage,
})
