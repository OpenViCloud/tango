import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"
import { ChevronDownIcon, ChevronRightIcon } from "lucide-react"

import type { ContainerModel, CreateContainerModel, ImageModel, PullImageModel } from "@/@types/models"
import { createContainerSchema, pullImageSchema } from "@/@types/models/container"
import {
  useGetContainerList,
  useGetImageList,
  useStartContainer,
  useStopContainer,
  useRemoveContainer,
  useCreateContainer,
  usePullImage,
  useRemoveImage,
} from "@/hooks/api/use-container"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { appIcons, actionIcons } from "@/lib/icons"

const DockerIcon = appIcons.containers
const CreateIcon = actionIcons.create
const StartIcon = actionIcons.start
const StopIcon = actionIcons.stop
const DeleteIcon = actionIcons.delete
const RefreshIcon = actionIcons.refresh

// ── State dot ─────────────────────────────────────────────────────────────────

const STATE_DOT: Record<string, string> = {
  running: "bg-green-500",
  created: "bg-blue-500",
  paused: "bg-yellow-500",
  restarting: "bg-yellow-500",
  dead: "bg-destructive",
}

function StateDot({ state }: { state: string }) {
  return (
    <span
      className={`inline-block size-2 shrink-0 rounded-full ${STATE_DOT[state] ?? "bg-muted-foreground/40"}`}
    />
  )
}

// ── Grouping helpers ──────────────────────────────────────────────────────────

type ContainerGroup = { project: string; containers: ContainerModel[] }

function groupContainers(containers: ContainerModel[]): {
  groups: ContainerGroup[]
  standalone: ContainerModel[]
} {
  const map = new Map<string, ContainerModel[]>()
  const standalone: ContainerModel[] = []

  for (const ct of containers) {
    const project = ct.labels?.["com.docker.compose.project"]
    if (project) {
      if (!map.has(project)) map.set(project, [])
      map.get(project)!.push(ct)
    } else {
      standalone.push(ct)
    }
  }

  return {
    groups: Array.from(map.entries()).map(([project, cts]) => ({ project, containers: cts })),
    standalone,
  }
}

// ── Table layout constant ─────────────────────────────────────────────────────

const COLS = "grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)_minmax(0,1.5fr)_9rem]"

// ── Table header ──────────────────────────────────────────────────────────────

function ContainerTableHeader() {
  const { t } = useTranslation()
  return (
    <div className={`${COLS} gap-4 px-4 py-2 text-xs font-medium text-muted-foreground border-b`}>
      <span>{t("docker.container.col.name")}</span>
      <span>{t("docker.container.col.image")}</span>
      <span>{t("docker.container.col.ports")}</span>
      <span className="text-right">{t("docker.container.col.actions")}</span>
    </div>
  )
}

// ── Single container row ──────────────────────────────────────────────────────

function ContainerRow({
  container,
  onStart,
  onStop,
  onRemove,
  busy,
  indent = false,
}: {
  container: ContainerModel
  onStart: (id: string) => void
  onStop: (id: string) => void
  onRemove: (id: string) => void
  busy: boolean
  indent?: boolean
}) {
  const isRunning = container.state === "running"

  const portSummary = container.ports
    .filter((p) => p.public_port > 0)
    .map((p) => `${p.public_port}→${p.private_port}`)
    .join(", ")

  return (
    <div
      className={`${COLS} gap-4 items-center px-4 py-2.5 border-b last:border-0 hover:bg-muted/30`}
    >
      <div className={`flex items-center gap-2 min-w-0 ${indent ? "pl-5" : ""}`}>
        <StateDot state={container.state} />
        <span className="font-mono text-sm truncate">
          {container.name || container.short_id}
        </span>
      </div>
      <span className="text-xs text-muted-foreground truncate">{container.image}</span>
      <span className="text-xs font-mono text-muted-foreground truncate">
        {portSummary || "—"}
      </span>
      <div className="flex items-center gap-1 justify-end">
        {isRunning ? (
          <Button variant="outline" size="sm" disabled={busy} onClick={() => onStop(container.id)}>
            <StopIcon className="size-3.5" />
          </Button>
        ) : (
          <Button variant="outline" size="sm" disabled={busy} onClick={() => onStart(container.id)}>
            <StartIcon className="size-3.5" />
          </Button>
        )}
        <Button
          variant="ghost"
          size="sm"
          disabled={busy}
          onClick={() => onRemove(container.id)}
          className="text-destructive hover:text-destructive"
        >
          <DeleteIcon className="size-3.5" />
        </Button>
      </div>
    </div>
  )
}

// ── Compose project group row ─────────────────────────────────────────────────

function ProjectGroupRow({
  group,
  onStart,
  onStop,
  onRemove,
  busy,
}: {
  group: ContainerGroup
  onStart: (id: string) => void
  onStop: (id: string) => void
  onRemove: (id: string) => void
  busy: boolean
}) {
  const [open, setOpen] = useState(true)
  const runningCount = group.containers.filter((c) => c.state === "running").length

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={`${COLS} w-full gap-4 items-center px-4 py-2.5 border-b hover:bg-muted/30 text-left`}
      >
        <div className="flex items-center gap-2">
          {open ? (
            <ChevronDownIcon className="size-3.5 text-muted-foreground shrink-0" />
          ) : (
            <ChevronRightIcon className="size-3.5 text-muted-foreground shrink-0" />
          )}
          <span className="text-sm font-medium">{group.project}</span>
          <span className="text-xs text-muted-foreground">
            {runningCount}/{group.containers.length}
          </span>
        </div>
        <span />
        <span />
        <span />
      </button>
      {open &&
        group.containers.map((ct) => (
          <ContainerRow
            key={ct.id}
            container={ct}
            onStart={onStart}
            onStop={onStop}
            onRemove={onRemove}
            busy={busy}
            indent
          />
        ))}
    </div>
  )
}

// ── Containers tab ────────────────────────────────────────────────────────────

function ContainersTab() {
  const { t } = useTranslation()
  const [showAll, setShowAll] = useState(false)
  const [showCreate, setShowCreate] = useState(false)

  const { data: containers, isLoading } = useGetContainerList(showAll)
  const startMutation = useStartContainer()
  const stopMutation = useStopContainer()
  const removeMutation = useRemoveContainer()

  const handleStart = (id: string) => {
    startMutation.mutate(id, {
      onSuccess: () => toast.success(t("docker.container.started")),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleStop = (id: string) => {
    stopMutation.mutate(id, {
      onSuccess: () => toast.success(t("docker.container.stopped")),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleRemove = (id: string) => {
    removeMutation.mutate(
      { id, force: false },
      {
        onSuccess: () => toast.success(t("docker.container.removed")),
        onError: (err) => toast.error(err.message),
      }
    )
  }

  const busy = startMutation.isPending || stopMutation.isPending || removeMutation.isPending
  const { groups, standalone } = groupContainers(containers ?? [])
  const isEmpty = (containers ?? []).length === 0

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <Button variant="outline" size="sm" onClick={() => setShowAll((v) => !v)}>
          {showAll ? t("docker.container.hideStoppedBtn") : t("docker.container.showAllBtn")}
        </Button>
        <Button size="sm" onClick={() => setShowCreate(true)}>
          <CreateIcon data-icon="inline-start" />
          {t("docker.container.createBtn")}
        </Button>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full rounded-lg" />
          ))}
        </div>
      ) : isEmpty ? (
        <p className="text-sm text-muted-foreground">{t("docker.container.empty")}</p>
      ) : (
        <div className="rounded-lg border overflow-hidden">
          <ContainerTableHeader />
          {standalone.map((ct) => (
            <ContainerRow
              key={ct.id}
              container={ct}
              onStart={handleStart}
              onStop={handleStop}
              onRemove={handleRemove}
              busy={busy}
            />
          ))}
          {groups.map((g) => (
            <ProjectGroupRow
              key={g.project}
              group={g}
              onStart={handleStart}
              onStop={handleStop}
              onRemove={handleRemove}
              busy={busy}
            />
          ))}
        </div>
      )}

      <CreateContainerSheet open={showCreate} onOpenChange={setShowCreate} />
    </div>
  )
}

// ── Create container sheet ────────────────────────────────────────────────────

function CreateContainerSheet({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { t } = useTranslation()
  const createMutation = useCreateContainer()

  const form = useForm<CreateContainerModel>({
    resolver: zodResolver(createContainerSchema),
    defaultValues: { image: "", name: "" },
  })

  const onSubmit = form.handleSubmit((values) => {
    createMutation.mutate(values, {
      onSuccess: () => {
        toast.success(t("docker.container.created"))
        form.reset()
        onOpenChange(false)
      },
      onError: (err) => toast.error(err.message),
    })
  })

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>{t("docker.container.createTitle")}</SheetTitle>
        </SheetHeader>
        <form onSubmit={onSubmit} className="flex flex-col gap-4 mt-4">
          <div className="flex flex-col gap-1.5">
            <Label>{t("docker.container.imageLabel")}</Label>
            <Input placeholder="nginx:latest" {...form.register("image")} />
            {form.formState.errors.image && (
              <p className="text-xs text-destructive">{form.formState.errors.image.message}</p>
            )}
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>{t("docker.container.nameLabel")}</Label>
            <Input
              placeholder={t("docker.container.namePlaceholder")}
              {...form.register("name")}
            />
          </div>
          <Button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending ? t("docker.actions.creating") : t("docker.actions.create")}
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ── Images tab ────────────────────────────────────────────────────────────────

function ImagesTab() {
  const { t } = useTranslation()
  const [showPull, setShowPull] = useState(false)

  const { data: images, isLoading } = useGetImageList()
  const removeMutation = useRemoveImage()

  const handleRemove = (id: string) => {
    removeMutation.mutate(
      { id, force: false },
      {
        onSuccess: () => toast.success(t("docker.image.removed")),
        onError: (err) => toast.error(err.message),
      }
    )
  }

  const isEmpty = (images ?? []).length === 0

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <Button size="sm" onClick={() => setShowPull(true)}>
          <RefreshIcon data-icon="inline-start" />
          {t("docker.image.pullBtn")}
        </Button>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full rounded-lg" />
          ))}
        </div>
      ) : isEmpty ? (
        <p className="text-sm text-muted-foreground">{t("docker.image.empty")}</p>
      ) : (
        <div className="rounded-lg border overflow-hidden">
          <ImageTableHeader />
          {(images ?? []).map((img) => (
            <ImageRow
              key={img.id}
              image={img}
              onRemove={handleRemove}
              busy={removeMutation.isPending}
            />
          ))}
        </div>
      )}

      <PullImageSheet open={showPull} onOpenChange={setShowPull} />
    </div>
  )
}

// ── Image table header ────────────────────────────────────────────────────────

const IMG_COLS = "grid grid-cols-[minmax(0,3fr)_minmax(0,1fr)_minmax(0,1fr)_6rem]"

function ImageTableHeader() {
  const { t } = useTranslation()
  return (
    <div className={`${IMG_COLS} gap-4 px-4 py-2 text-xs font-medium text-muted-foreground border-b`}>
      <span>{t("docker.image.col.tag")}</span>
      <span>{t("docker.image.col.id")}</span>
      <span>{t("docker.image.col.size")}</span>
      <span className="text-right">{t("docker.image.col.actions")}</span>
    </div>
  )
}

function ImageRow({
  image,
  onRemove,
  busy,
}: {
  image: ImageModel
  onRemove: (id: string) => void
  busy: boolean
}) {
  const { t } = useTranslation()
  const tag = image.tags[0] ?? "<none>"

  return (
    <div className={`${IMG_COLS} gap-4 items-center px-4 py-2.5 border-b last:border-0 hover:bg-muted/30`}>
      <div className="flex items-center gap-2 min-w-0">
        <span className="font-mono text-sm truncate">{tag}</span>
        {image.in_use > 0 && (
          <Badge variant="secondary" className="shrink-0">
            {t("docker.image.inUse", { count: image.in_use })}
          </Badge>
        )}
      </div>
      <span className="text-xs text-muted-foreground font-mono">{image.short_id}</span>
      <span className="text-xs text-muted-foreground">{image.size}</span>
      <div className="flex justify-end">
        <Button
          variant="ghost"
          size="sm"
          disabled={busy || image.in_use > 0}
          onClick={() => onRemove(image.id)}
          className="text-destructive hover:text-destructive"
        >
          <DeleteIcon className="size-3.5" />
        </Button>
      </div>
    </div>
  )
}

function PullImageSheet({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { t } = useTranslation()
  const pullMutation = usePullImage()

  const form = useForm<PullImageModel>({
    resolver: zodResolver(pullImageSchema),
    defaultValues: { reference: "" },
  })

  const onSubmit = form.handleSubmit((values) => {
    pullMutation.mutate(values, {
      onSuccess: () => {
        toast.success(t("docker.image.pulled"))
        form.reset()
        onOpenChange(false)
      },
      onError: (err) => toast.error(err.message),
    })
  })

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>{t("docker.image.pullTitle")}</SheetTitle>
        </SheetHeader>
        <form onSubmit={onSubmit} className="flex flex-col gap-4 mt-4">
          <div className="flex flex-col gap-1.5">
            <Label>{t("docker.image.referenceLabel")}</Label>
            <Input placeholder="nginx:latest" {...form.register("reference")} />
            {form.formState.errors.reference && (
              <p className="text-xs text-destructive">
                {form.formState.errors.reference.message}
              </p>
            )}
          </div>
          <Button type="submit" disabled={pullMutation.isPending}>
            {pullMutation.isPending ? t("docker.actions.pulling") : t("docker.actions.pull")}
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export function ContainersPage() {
  const { t } = useTranslation()

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<DockerIcon />}
        title={t("docker.page.title")}
        description={t("docker.page.description")}
      />
      <SectionCard>
        <Tabs defaultValue="containers">
          <TabsList>
            <TabsTrigger value="containers">{t("docker.tabs.containers")}</TabsTrigger>
            <TabsTrigger value="images">{t("docker.tabs.images")}</TabsTrigger>
          </TabsList>
          <TabsContent value="containers" className="mt-4">
            <ContainersTab />
          </TabsContent>
          <TabsContent value="images" className="mt-4">
            <ImagesTab />
          </TabsContent>
        </Tabs>
      </SectionCard>
    </div>
  )
}
