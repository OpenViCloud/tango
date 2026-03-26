import {
  Outlet,
  createFileRoute,
  redirect,
  useRouterState,
} from "@tanstack/react-router"

import { AppShell } from "@/components/app-shell"
import { ThemeToggle } from "@/components/theme-toggle"
import { IdentityShellActions } from "@/routes/_auth/users/components/-identity-shell-actions"
import { useAuthStore } from "@/store/auth"
import { useTranslation } from "react-i18next"

function AuthLayout() {
  const { t } = useTranslation()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  let section = t("shell.workspace")
  let title = t("dashboard.title")
  let actions: React.ReactNode = <ThemeToggle />

  if (pathname === "/users") {
    section = t("shell.identity")
    title = t("identity.usersTitle")
    actions = <IdentityShellActions />
  } else if (pathname === "/users/create") {
    section = t("shell.identity")
    title = t("identity.createPageTitle")
    actions = <IdentityShellActions />
  } else if (pathname.startsWith("/users/") && pathname.endsWith("/edit")) {
    section = t("shell.identity")
    title = t("identity.editPageTitle")
    actions = <IdentityShellActions />
  } else if (pathname === "/roles") {
    section = t("shell.identity")
    title = t("identity.rolesTitle")
    actions = <IdentityShellActions />
  } else if (pathname === "/channels") {
    section = t("shell.workspace")
    title = t("channels.listPageTitle")
  } else if (pathname === "/channels/create") {
    section = t("shell.workspace")
    title = t("channels.createPageTitle")
  } else if (pathname.startsWith("/channels/") && pathname.endsWith("/edit")) {
    section = t("shell.workspace")
    title = t("channels.editPageTitle")
  } else if (pathname === "/builds") {
    section = t("builds.section")
    title = t("builds.page.title")
  } else if (pathname === "/containers") {
    section = t("shell.workspace")
    title = t("docker.page.title")
  }

  return (
    <AppShell section={section} title={title} actions={actions}>
      <Outlet />
    </AppShell>
  )
}

export const Route = createFileRoute("/_auth")({
  beforeLoad: () => {
    const token = useAuthStore.getState().accessToken
    if (!token) {
      throw redirect({ to: "/login" })
    }
  },
  component: AuthLayout,
})
