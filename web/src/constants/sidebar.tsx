import type { ComponentType } from "react"

import { appIcons } from "@/lib/icons"
export type SidebarChildItem = {
  titleKey: string
  url: string
}

export type SidebarItem = {
  titleKey: string
  url: string
  icon?: ComponentType<{ className?: string }>
  isActive?: boolean
  children?: SidebarChildItem[]
}

export type SidebarSection = {
  labelKey: string
  items: SidebarItem[]
}

export const appSidebarSections: SidebarSection[] = [
  {
    labelKey: "sidebar.platform",
    items: [
      {
        titleKey: "common.dashboard",
        url: "/dashboard",
        icon: appIcons.dashboard,
      },
      {
        titleKey: "sidebar.projects",
        url: "/projects",
        icon: appIcons.projects,
      },
      {
        titleKey: "sidebar.containers",
        url: "/containers",
        icon: appIcons.docker,
      },
      {
        titleKey: "sidebar.swarm",
        url: "/swarm",
        icon: appIcons.swarm,
      },
      {
        titleKey: "sidebar.servers",
        url: "/servers",
        icon: appIcons.servers,
      },
      {
        titleKey: "sidebar.clusters",
        url: "/clusters",
        icon: appIcons.clusters,
      },
    ],
  },
  {
    labelKey: "sidebar.settings",
    items: [
      {
        titleKey: "sidebar.settingsGeneral",
        url: "/settings",
        icon: appIcons.settings,
      },
      {
        titleKey: "sidebar.sources",
        url: "/sources",
        icon: appIcons.sources,
      },
      {
        titleKey: "sidebar.domains",
        url: "/domains",
        icon: appIcons.domains,
      },
      {
        titleKey: "sidebar.databases",
        url: "/databases",
        icon: appIcons.databases,
      },

      {
        titleKey: "sidebar.builds",
        url: "/builds",
        icon: appIcons.builds,
      },
      {
        titleKey: "sidebar.channels",
        url: "/channels",
        icon: appIcons.channels,
      },
    ],
  },
  {
    labelKey: "sidebar.identity",
    items: [
      {
        titleKey: "sidebar.users",
        url: "/users",
        icon: appIcons.users,
      },
      {
        titleKey: "sidebar.roles",
        url: "/roles",
        icon: appIcons.roles,
      },
    ],
  },
]
