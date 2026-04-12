import { createFileRoute } from "@tanstack/react-router"

import { CloudflarePage } from "@/pages/auth/cloudflare/cloudflare-page"

export const Route = createFileRoute("/_auth/cloudflare/")({
  component: CloudflarePage,
})
