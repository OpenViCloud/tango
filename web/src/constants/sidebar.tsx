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
        titleKey: "sidebar.workspace",
        url: "/dashboard",
        icon: appIcons.dashboard,
        isActive: true,
        children: [{ titleKey: "sidebar.overview", url: "/dashboard" }],
      },
      {
        titleKey: "sidebar.channels",
        url: "/channels",
        icon: appIcons.channels,
        children: [{ titleKey: "sidebar.channelList", url: "/channels" }],
      },
      {
        titleKey: "sidebar.builds",
        url: "/builds",
        icon: appIcons.builds,
        children: [{ titleKey: "sidebar.buildList", url: "/builds" }],
      },
      {
        titleKey: "sidebar.containers",
        url: "/containers",
        icon: appIcons.containers,
        children: [{ titleKey: "sidebar.containerList", url: "/containers" }],
      },
      {
        titleKey: "sidebar.projects",
        url: "/projects",
        icon: appIcons.projects,
        children: [{ titleKey: "sidebar.projectList", url: "/projects" }],
      },
      {
        titleKey: "sidebar.sources",
        url: "/sources",
        icon: appIcons.sources,
        children: [{ titleKey: "sidebar.sourceList", url: "/sources" }],
      },
      {
        titleKey: "sidebar.databases",
        url: "/databases",
        icon: appIcons.databases,
        children: [{ titleKey: "sidebar.databaseList", url: "/databases" }],
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
        children: [{ titleKey: "sidebar.userList", url: "/users" }],
      },
      {
        titleKey: "sidebar.roles",
        url: "/roles",
        icon: appIcons.roles,
        children: [{ titleKey: "sidebar.roleList", url: "/roles" }],
      },
    ],
  },
]
