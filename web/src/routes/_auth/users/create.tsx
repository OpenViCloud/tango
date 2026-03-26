import { createFileRoute } from "@tanstack/react-router"

import UsersCreatePage from "@/pages/auth/users/users-create-page"

export const Route = createFileRoute("/_auth/users/create")({
  component: UsersCreatePage,
})
