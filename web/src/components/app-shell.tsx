import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"

import { AppSidebar } from "@/components/app-sidebar"
import { LanguageSwitcher } from "@/components/language-switcher"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"

type AppShellProps = {
  section?: string
  title: string
  actions?: ReactNode
  children: ReactNode
}

export function AppShell({
  section = "Tango",
  title,
  actions,
  children,
}: AppShellProps) {
  const { t } = useTranslation()

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="sticky top-0 z-20 flex h-16 shrink-0 items-center justify-between gap-2 border-b border-border/90 bg-background/92 backdrop-blur-sm">
          <div className="flex w-full items-center justify-between gap-2 px-4">
            <div className="flex items-center gap-2">
              <SidebarTrigger className="-ml-1" />
              <Separator
                orientation="vertical"
                className="mr-2 data-[orientation=vertical]:h-4"
              />
              <Breadcrumb>
                <BreadcrumbList>
                  <BreadcrumbItem className="hidden md:block">
                    <BreadcrumbLink href="#">
                      {section === "Tango" ? t("shell.appName") : section}
                    </BreadcrumbLink>
                  </BreadcrumbItem>
                  <BreadcrumbSeparator className="hidden md:block" />
                  <BreadcrumbItem>
                    <BreadcrumbPage>{title}</BreadcrumbPage>
                  </BreadcrumbItem>
                </BreadcrumbList>
              </Breadcrumb>
            </div>
            <div className="flex items-center gap-2">
              <LanguageSwitcher />
              {actions}
            </div>
          </div>
        </header>
        <div className="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-6 p-4">
          {children}
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
