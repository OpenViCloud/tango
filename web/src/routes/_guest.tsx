import { Outlet, createFileRoute, redirect } from "@tanstack/react-router"

import { useAuthStore } from "@/store/auth"

export const Route = createFileRoute("/_guest")({
  beforeLoad: () => {
    const token = useAuthStore.getState().accessToken
    if (token) {
      throw redirect({ to: "/dashboard" })
    }
  },
  component: () => <Outlet />,
})
