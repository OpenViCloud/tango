import { createFileRoute } from "@tanstack/react-router"

import UsersEditPage from "@/pages/auth/users/users-edit-page"

function UsersEditRoute() {
  const { userId } = Route.useParams()

  return <UsersEditPage userId={userId} />
}

export const Route = createFileRoute("/_auth/users/$userId/edit")({
  component: UsersEditRoute,
})
