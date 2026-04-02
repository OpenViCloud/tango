import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { useSidebar } from "@/components/ui/sidebar-context"
import { useGetCurrentUser } from "@/hooks/api/use-user"
import { useAuthStore } from "@/store/auth"
import { useNavigate } from "@tanstack/react-router"
import {
  BadgeCheckIcon,
  ChevronsUpDownIcon,
  LogOutIcon,
} from "lucide-react"
import { useTranslation } from "react-i18next"

function getDisplayName(user?: {
  nickname?: string
  first_name?: string
  last_name?: string
  email?: string
}) {
  const fullName = [user?.first_name, user?.last_name]
    .filter(Boolean)
    .join(" ")
    .trim()

  if (fullName) return fullName
  if (user?.nickname?.trim()) return user.nickname.trim()
  return user?.email ?? "Account"
}

function getInitials(name: string) {
  const tokens = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)

  if (tokens.length === 0) return "AC"

  return tokens.map((token) => token[0]?.toUpperCase() ?? "").join("")
}

export function NavUser() {
  const { t } = useTranslation()
  const { isMobile } = useSidebar()
  const { logout } = useAuthStore()
  const navigate = useNavigate()
  const { data: user, isLoading } = useGetCurrentUser()

  const handleLogout = async () => {
    await logout()
    navigate({ to: "/login" })
  }

  if (isLoading) {
    return <div>{t("userMenu.loading")}</div>
  }

  const username = getDisplayName(user)
  const initials = getInitials(username)

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <Avatar className="h-8 w-8 rounded-lg">
                <AvatarImage alt={username} />
                <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium">{username}</span>
                <span className="truncate text-xs">{user?.email}</span>
              </div>
              <ChevronsUpDownIcon className="ml-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <Avatar className="h-8 w-8 rounded-lg">
                  <AvatarImage alt={username} />
                  <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">{username}</span>
                  <span className="truncate text-xs">{user?.email}</span>
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => navigate({ to: "/account" })}>
                <BadgeCheckIcon />
                {t("userMenu.account")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleLogout}>
              <LogOutIcon />
              {t("userMenu.logout")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
