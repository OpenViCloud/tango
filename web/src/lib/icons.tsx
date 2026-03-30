import type { ComponentProps, ComponentType } from "react"
import {
  ChatRoundDots,
  HomeSmile,
  ShieldCheck,
  UsersGroupRounded,
} from "@solar-icons/react"
import {
  ArrowLeftIcon,
  BoxIcon,
  ContainerIcon,
  CopyIcon,
  DatabaseIcon,
  FilterIcon,
  FolderIcon,
  FolderGit2Icon,
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
  builds: BoxIcon,
  containers: ContainerIcon,
  databases: DatabaseIcon,
  projects: FolderIcon,
  sources: FolderGit2Icon,
  settings: Settings2Icon,
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
