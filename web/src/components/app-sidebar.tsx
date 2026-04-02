"use client"

import { useLocation } from "@tanstack/react-router"
import * as React from "react"

import { NavMain } from "@/components/nav-main"
import { NavUser } from "@/components/nav-user"
import { TeamSwitcher } from "@/components/team-switcher"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar"
import { useSidebar } from "@/components/ui/sidebar-context"
import { appSidebarSections } from "@/constants/sidebar"
import LogoIcon from "@/icons/logo-icon"
import { AudioLinesIcon, TerminalIcon } from "lucide-react"

// This is sample data.
const data = {
  user: {
    name: "shadcn",
    email: "m@example.com",
    avatar: "/avatars/shadcn.jpg",
  },
  teams: [
    {
      name: "Tango Cloud",
      logo: <LogoIcon className="size-6 text-sidebar-primary" />,
      plan: "Enterprise",
    },
    {
      name: "Acme Corp.",
      logo: <AudioLinesIcon />,
      plan: "Startup",
    },
    {
      name: "Evil Corp.",
      logo: <TerminalIcon />,
      plan: "Free",
    },
  ],
}

function SidebarNavigationSync() {
  const location = useLocation()
  const { isMobile, openMobile, setOpenMobile } = useSidebar()
  const previousPathnameRef = React.useRef(location.pathname)

  React.useEffect(() => {
    if (
      isMobile &&
      openMobile &&
      previousPathnameRef.current !== location.pathname
    ) {
      setOpenMobile(false)
    }

    previousPathnameRef.current = location.pathname
  }, [isMobile, location.pathname, openMobile, setOpenMobile])

  return null
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarNavigationSync />
      <SidebarHeader>
        <TeamSwitcher teams={data.teams} />
      </SidebarHeader>
      <SidebarContent>
        <NavMain sections={appSidebarSections} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
