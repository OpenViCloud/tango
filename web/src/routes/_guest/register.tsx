import { createFileRoute } from "@tanstack/react-router"

import RegisterPage from "@/pages/guest/register-page"

export const Route = createFileRoute("/_guest/register")({
  component: RegisterPage,
})
