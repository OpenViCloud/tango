import { createFileRoute } from "@tanstack/react-router"

import LoginPage from "@/pages/guest/login-page"

export const Route = createFileRoute("/_guest/login")({
  component: LoginPage,
})
