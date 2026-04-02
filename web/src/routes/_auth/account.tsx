import { createFileRoute } from "@tanstack/react-router"

import { AccountPage } from "@/pages/auth/account/account-page"

export const Route = createFileRoute("/_auth/account")({
  component: AccountPage,
})
