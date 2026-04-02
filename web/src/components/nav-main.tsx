import type { SidebarSection } from "@/constants/sidebar"
import { Link, useLocation } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar"
import { useSidebar } from "@/components/ui/sidebar-context"
import { ChevronRightIcon } from "lucide-react"

export function NavMain({ sections }: { sections: SidebarSection[] }) {
  const { t } = useTranslation()
  const location = useLocation()
  const { isMobile, setOpenMobile } = useSidebar()

  const isRouteActive = (url: string) =>
    location.pathname === url || location.pathname.startsWith(`${url}/`)

  return (
    <>
      {sections.map((section) => (
        <SidebarGroup key={section.labelKey}>
          <SidebarGroupLabel>{t(section.labelKey)}</SidebarGroupLabel>
          <SidebarMenu>
            {section.items.map((item) => {
              const isItemActive =
                isRouteActive(item.url) ||
                item.children?.some((subItem) => isRouteActive(subItem.url)) ||
                false
              const Icon = item.icon
              const hasChildren = Boolean(item.children?.length)

              if (!hasChildren) {
                return (
                  <SidebarMenuItem key={item.titleKey}>
                    <SidebarMenuButton
                      asChild
                      isActive={isItemActive}
                      size="lg"
                      tooltip={t(item.titleKey)}
                      className="group-data-[collapsible=icon]:size-9 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:[&_svg]:size-5"
                    >
                      <Link
                        to={item.url}
                        onClick={() => {
                          if (isMobile) {
                            setOpenMobile(false)
                          }
                        }}
                      >
                        {Icon ? <Icon className="size-6" /> : null}
                        <span className="group-data-[collapsible=icon]:hidden">
                          {t(item.titleKey)}
                        </span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              }

              return (
                <Collapsible
                  key={item.titleKey}
                  asChild
                  defaultOpen={isItemActive || item.isActive}
                  className="group/collapsible"
                >
                  <SidebarMenuItem>
                  <CollapsibleTrigger asChild>
                    <SidebarMenuButton
                      isActive={isItemActive}
                      size="lg"
                      tooltip={t(item.titleKey)}
                      className="group-data-[collapsible=icon]:size-9 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:[&_svg]:size-5"
                    >
                      {Icon ? <Icon className="size-6" /> : null}
                      <span className="group-data-[collapsible=icon]:hidden">
                        {t(item.titleKey)}
                      </span>
                      <ChevronRightIcon className="ml-auto transition-transform duration-200 group-data-[collapsible=icon]:hidden group-data-[state=open]/collapsible:rotate-90" />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <SidebarMenuSub>
                      {item.children?.map((subItem) => {
                        const isSubItemActive = location.pathname === subItem.url

                        return (
                          <SidebarMenuSubItem key={subItem.titleKey}>
                        <SidebarMenuSubButton asChild isActive={isSubItemActive}>
                            <Link
                              to={subItem.url}
                              onClick={() => {
                                if (isMobile) {
                                  setOpenMobile(false)
                                }
                              }}
                            >
                              <span>{t(subItem.titleKey)}</span>
                            </Link>
                          </SidebarMenuSubButton>
                        </SidebarMenuSubItem>
                        )
                      })}
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>
              )
            })}
          </SidebarMenu>
        </SidebarGroup>
      ))}
    </>
  )
}
