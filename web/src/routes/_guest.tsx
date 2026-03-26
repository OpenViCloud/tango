import { Outlet, createFileRoute, redirect } from "@tanstack/react-router"

import { useAuthStore } from "@/store/auth"

export const Route = createFileRoute("/_guest")({
  beforeLoad: () => {
    const { isAuthenticated } = useAuthStore.getState()
    if (isAuthenticated) {
      throw redirect({ to: "/dashboard" })
    }
  },
  component: () => <Outlet />,
})
