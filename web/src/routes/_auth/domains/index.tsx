import { createFileRoute } from "@tanstack/react-router"

import { DomainsPage } from "@/pages/auth/domains/domains-page"

export const Route = createFileRoute("/_auth/domains/")({
  component: DomainsPage,
})
