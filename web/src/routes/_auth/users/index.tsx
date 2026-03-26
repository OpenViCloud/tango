import { createFileRoute } from "@tanstack/react-router"

import UsersListPage from "@/pages/auth/users/users-list-page"

export const Route = createFileRoute("/_auth/users/")({
  component: UsersListPage,
})
