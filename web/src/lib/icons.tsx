import DockerIcon from "@/icons/docker-icon"
import GitIcon from "@/icons/git-icon"
import {
  Box,
  ChatRoundDots,
  Database,
  Global,
  HomeSmile,
  Lightning,
  Settings,
  ShieldCheck,
  UsersGroupRounded,
} from "@solar-icons/react"
import {
  ArrowLeftIcon,
  CopyIcon,
  FilterIcon,
  PlayIcon,
  PlusIcon,
  RefreshCwIcon,
  SearchIcon,
  Settings2Icon,
  SquareIcon,
  SquarePenIcon,
  Trash2Icon,
  UploadIcon,
} from "lucide-react"
import type { ComponentProps, ComponentType } from "react"

type SolarIconProps = ComponentProps<typeof HomeSmile>

function createSolarIcon(Icon: ComponentType<SolarIconProps>) {
  return function AppSolarIcon({
    className = "size-6",
    weight = "BoldDuotone",
    ...props
  }: SolarIconProps) {
    return <Icon className={className} weight={weight} {...props} />
  }
}

export const appIcons = {
  builds: createSolarIcon(Lightning),
  docker: DockerIcon,
  databases: createSolarIcon(Database),
  domains: createSolarIcon(Global),
  projects: createSolarIcon(Box),
  sources: GitIcon,
  settings: createSolarIcon(Settings),
  dashboard: createSolarIcon(HomeSmile),
  channels: createSolarIcon(ChatRoundDots),
  users: createSolarIcon(UsersGroupRounded),
  roles: createSolarIcon(ShieldCheck),
} as const

export const actionIcons = {
  back: ArrowLeftIcon,
  copy: CopyIcon,
  create: PlusIcon,
  delete: Trash2Icon,
  edit: SquarePenIcon,
  filter: FilterIcon,
  refresh: RefreshCwIcon,
  restart: RefreshCwIcon,
  search: SearchIcon,
  settings: Settings2Icon,
  start: PlayIcon,
  stop: SquareIcon,
  upload: UploadIcon,
} as const
