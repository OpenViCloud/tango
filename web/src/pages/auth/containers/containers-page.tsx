import { useEffect, useRef, useState } from "react"
import { useForm, useFieldArray } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"
import {
  ChevronDownIcon,
  ChevronRightIcon,
  PlusIcon,
  MinusIcon,
} from "lucide-react"
import { useQueryClient } from "@tanstack/react-query"

import type {
  ContainerModel,
  CreateContainerModel,
  ImageModel,
  PullImageModel,
} from "@/@types/models"
import { pullImageSchema } from "@/@types/models/container"
import {
  useGetContainer,
  useGetContainerList,
  useGetContainerStats,
  useGetImageList,
  useStartContainer,
  useStopContainer,
  useRemoveContainer,
  useCreateContainer,
  useRemoveImage,
  CONTAINER_QUERY_KEYS,
} from "@/hooks/api/use-container"
import { usePullImageLogs } from "@/hooks/api/use-pull-image-logs"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetFooter,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { appIcons, actionIcons } from "@/lib/icons"

const DockerIcon = appIcons.docker
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

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  let size = value
  let unitIndex = 0
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }
  return `${size.toFixed(size >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`
}

function formatDateTime(value?: string) {
  if (!value) return "—"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "—"
  return new Intl.DateTimeFormat("vi-VN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date)
}

function statBadgeVariant(
  state: string
): "default" | "secondary" | "outline" | "warning" | "success" {
  if (state === "running") return "success"
  if (state === "paused" || state === "restarting") return "warning"
  if (state === "created") return "default"
  return "secondary"
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
    groups: Array.from(map.entries()).map(([project, cts]) => ({
      project,
      containers: cts,
    })),
    standalone,
  }
}

// ── Table layout constant ─────────────────────────────────────────────────────

const COLS =
  "grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)_minmax(0,1.5fr)_14rem]"

// ── Table header ──────────────────────────────────────────────────────────────

function ContainerTableHeader() {
  const { t } = useTranslation()
  return (
    <div
      className={`${COLS} gap-4 border-b px-4 py-2 text-xs font-medium text-muted-foreground`}
    >
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
  onDetails,
  onStart,
  onStop,
  onRemove,
  busy,
  indent = false,
}: {
  container: ContainerModel
  onDetails: (id: string) => void
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
      className={`${COLS} items-center gap-4 border-b px-4 py-2.5 last:border-0 hover:bg-muted/30`}
    >
      <div
        className={`flex min-w-0 items-center gap-2 ${indent ? "pl-5" : ""}`}
      >
        <StateDot state={container.state} />
        <span className="truncate font-mono text-sm">
          {container.name || container.short_id}
        </span>
      </div>
      <span className="truncate text-xs text-muted-foreground">
        {container.image}
      </span>
      <span className="truncate font-mono text-xs text-muted-foreground">
        {portSummary || "—"}
      </span>
      <div className="flex items-center justify-end gap-1">
        <Button
          variant="outline"
          size="sm"
          disabled={busy}
          onClick={() => onDetails(container.id)}
        >
          Details
        </Button>
        {isRunning ? (
          <Button
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={() => onStop(container.id)}
          >
            <StopIcon className="size-3.5" />
          </Button>
        ) : (
          <Button
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={() => onStart(container.id)}
          >
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
  onDetails,
  onStart,
  onStop,
  onRemove,
  busy,
}: {
  group: ContainerGroup
  onDetails: (id: string) => void
  onStart: (id: string) => void
  onStop: (id: string) => void
  onRemove: (id: string) => void
  busy: boolean
}) {
  const [open, setOpen] = useState(true)
  const runningCount = group.containers.filter(
    (c) => c.state === "running"
  ).length

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={`${COLS} w-full items-center gap-4 border-b px-4 py-2.5 text-left hover:bg-muted/30`}
      >
        <div className="flex items-center gap-2">
          {open ? (
            <ChevronDownIcon className="size-3.5 shrink-0 text-muted-foreground" />
          ) : (
            <ChevronRightIcon className="size-3.5 shrink-0 text-muted-foreground" />
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
            onDetails={onDetails}
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

function DetailMetric({
  label,
  value,
  hint,
}: {
  label: string
  value: string
  hint?: string
}) {
  return (
    <div className="rounded-xl border border-border/70 bg-background/70 p-4">
      <p className="text-xs tracking-[0.16em] text-muted-foreground uppercase">
        {label}
      </p>
      <p className="mt-2 text-lg font-semibold">{value}</p>
      {hint ? (
        <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
      ) : null}
    </div>
  )
}

function ContainerDetailsSheet({
  containerId,
  open,
  onOpenChange,
}: {
  containerId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { data: details, isLoading: detailsLoading } = useGetContainer(
    containerId ?? ""
  )
  const { data: stats, isLoading: statsLoading } = useGetContainerStats(
    containerId ?? "",
    open
  )

  const labels = Object.entries(details?.labels ?? {})
  const networks = Object.entries(details?.networks ?? {})

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="flex flex-col gap-0 overflow-y-auto sm:max-w-2xl">
        <SheetHeader className="border-b pb-4">
          <SheetTitle>
            {details?.name || details?.short_id || "Container details"}
          </SheetTitle>
          {details ? (
            <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
              <Badge variant={statBadgeVariant(details.state)}>
                {details.state}
              </Badge>
              <span className="font-mono">{details.image}</span>
            </div>
          ) : null}
        </SheetHeader>

        {detailsLoading ? (
          <div className="flex flex-col gap-4 py-4">
            <Skeleton className="h-24 rounded-xl" />
            <Skeleton className="h-24 rounded-xl" />
            <Skeleton className="h-56 rounded-xl" />
          </div>
        ) : !details ? (
          <div className="py-6 text-sm text-muted-foreground">
            Could not load container details.
          </div>
        ) : (
          <div className="flex flex-col gap-6 p-4">
            <div className="flex flex-col gap-3">
              <DetailMetric
                label="CPU"
                value={stats ? `${stats.cpu_percent.toFixed(1)}%` : "—"}
              />
              <DetailMetric
                label="Memory"
                value={
                  stats
                    ? `${formatBytes(stats.memory_usage_bytes)} / ${formatBytes(stats.memory_limit_bytes)}`
                    : "—"
                }
                hint={
                  stats ? `${stats.memory_percent.toFixed(1)}% used` : undefined
                }
              />
              <DetailMetric
                label="Network"
                value={
                  stats
                    ? `${formatBytes(stats.network_rx_bytes)} down / ${formatBytes(stats.network_tx_bytes)} up`
                    : "—"
                }
              />
              <DetailMetric
                label="Block I/O"
                value={
                  stats
                    ? `${formatBytes(stats.block_read_bytes)} read / ${formatBytes(stats.block_write_bytes)} write`
                    : "—"
                }
              />
              <DetailMetric
                label="PIDs"
                value={stats ? String(stats.pids_current) : "—"}
              />
              <DetailMetric
                label="Last sample"
                value={
                  stats
                    ? formatDateTime(stats.read_at)
                    : statsLoading
                      ? "Loading..."
                      : "—"
                }
              />
            </div>

            <div className="rounded-2xl border border-border/70 p-4">
              <h3 className="text-sm font-semibold">Container</h3>
              <div className="mt-4 grid gap-3 text-sm">
                <div>
                  <p className="text-muted-foreground">Created</p>
                  <p>{formatDateTime(details.created_at)}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Started</p>
                  <p>{formatDateTime(details.started_at)}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Finished</p>
                  <p>{formatDateTime(details.finished_at)}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Restart count</p>
                  <p>{details.restart_count}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Exit code</p>
                  <p>{details.exit_code}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Command</p>
                  <p className="font-mono text-xs break-all">
                    {details.command.length > 0
                      ? details.command.join(" ")
                      : "—"}
                  </p>
                </div>
                {details.error ? (
                  <div>
                    <p className="text-muted-foreground">Error</p>
                    <p className="text-destructive">{details.error}</p>
                  </div>
                ) : null}
              </div>
            </div>

            <div className="rounded-2xl border border-border/70 p-4">
              <h3 className="text-sm font-semibold">Networks</h3>
              <div className="mt-4 flex flex-col gap-3">
                {networks.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    No network addresses found.
                  </p>
                ) : (
                  networks.map(([name, ip]) => (
                    <div
                      key={name}
                      className="rounded-xl border border-border/70 bg-background/70 p-3"
                    >
                      <p className="font-medium">{name}</p>
                      <p className="mt-1 font-mono text-xs text-muted-foreground">
                        {ip || "—"}
                      </p>
                    </div>
                  ))
                )}
              </div>
            </div>

            <div className="rounded-2xl border border-border/70 p-4">
              <h3 className="text-sm font-semibold">Ports</h3>
              <div className="mt-4 flex flex-col gap-2">
                {details.ports.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    No ports exposed.
                  </p>
                ) : (
                  details.ports.map((port, index) => (
                    <div
                      key={`${port.private_port}-${port.public_port}-${index}`}
                      className="flex items-center justify-between rounded-xl border border-border/70 bg-background/70 px-3 py-2 text-sm"
                    >
                      <span className="font-mono">
                        {port.public_port
                          ? `${port.public_port} → ${port.private_port}`
                          : `${port.private_port}`}
                      </span>
                      <Badge variant="outline">{port.type}</Badge>
                    </div>
                  ))
                )}
              </div>
            </div>

            <div className="rounded-2xl border border-border/70 p-4">
              <h3 className="text-sm font-semibold">Mounts</h3>
              <div className="mt-4 flex flex-col gap-2">
                {details.mounts.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    No mounts configured.
                  </p>
                ) : (
                  details.mounts.map((mount, index) => (
                    <div
                      key={`${mount.destination}-${index}`}
                      className="rounded-xl border border-border/70 bg-background/70 p-3"
                    >
                      <div className="flex items-center justify-between gap-3">
                        <p className="font-mono text-xs">{mount.destination}</p>
                        <Badge variant="outline">{mount.type}</Badge>
                      </div>
                      <p className="mt-2 text-xs break-all text-muted-foreground">
                        {mount.source || mount.name || "—"}
                      </p>
                    </div>
                  ))
                )}
              </div>
            </div>

            <div className="rounded-2xl border border-border/70 p-4">
              <h3 className="text-sm font-semibold">Labels</h3>
              <div className="mt-4 flex flex-col gap-2">
                {labels.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    No labels found.
                  </p>
                ) : (
                  labels.map(([key, value]) => (
                    <div
                      key={key}
                      className="flex flex-col gap-1 rounded-xl border border-border/70 bg-background/70 p-3"
                    >
                      <p className="font-mono text-xs break-all">{key}</p>
                      <p className="text-xs break-all text-muted-foreground">
                        {value}
                      </p>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        )}
      </SheetContent>
    </Sheet>
  )
}

// ── Containers tab ────────────────────────────────────────────────────────────

function ContainersTab() {
  const { t } = useTranslation()
  const [showAll, setShowAll] = useState(false)
  const [showCreate, setShowCreate] = useState(false)
  const [selectedContainerId, setSelectedContainerId] = useState<string | null>(
    null
  )

  const { data: containers, isLoading } = useGetContainerList(showAll)
  const startMutation = useStartContainer()
  const stopMutation = useStopContainer()
  const removeMutation = useRemoveContainer()

  const handleStart = (id: string) => {
    startMutation.mutate(id, {
      onSuccess: () => toast.success(t("docker.container.started")),
    })
  }

  const handleStop = (id: string) => {
    stopMutation.mutate(id, {
      onSuccess: () => toast.success(t("docker.container.stopped")),
    })
  }

  const handleRemove = (id: string) => {
    removeMutation.mutate(
      { id, force: false },
      {
        onSuccess: () => toast.success(t("docker.container.removed")),
      }
    )
  }

  const handleDetails = (id: string) => {
    setSelectedContainerId(id)
  }

  const busy =
    startMutation.isPending ||
    stopMutation.isPending ||
    removeMutation.isPending
  const { groups, standalone } = groupContainers(containers ?? [])
  const isEmpty = (containers ?? []).length === 0

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setShowAll((v) => !v)}
        >
          {showAll
            ? t("docker.container.hideStoppedBtn")
            : t("docker.container.showAllBtn")}
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
        <p className="text-sm text-muted-foreground">
          {t("docker.container.empty")}
        </p>
      ) : (
        <div className="overflow-hidden rounded-lg border">
          <ContainerTableHeader />
          {standalone.map((ct) => (
            <ContainerRow
              key={ct.id}
              container={ct}
              onDetails={handleDetails}
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
              onDetails={handleDetails}
              onStart={handleStart}
              onStop={handleStop}
              onRemove={handleRemove}
              busy={busy}
            />
          ))}
        </div>
      )}

      <RunContainerSheet open={showCreate} onOpenChange={setShowCreate} />
      <ContainerDetailsSheet
        containerId={selectedContainerId}
        open={Boolean(selectedContainerId)}
        onOpenChange={(open) => {
          if (!open) setSelectedContainerId(null)
        }}
      />
    </div>
  )
}

// ── Run container sheet ───────────────────────────────────────────────────────

type PortRow = { hostPort: string; containerPort: string }
type VolumeRow = { hostPath: string; containerPath: string }
type EnvRow = { key: string; value: string }
type RunContainerForm = {
  image: string
  name: string
  ports: PortRow[]
  volumes: VolumeRow[]
  envVars: EnvRow[]
}

function RunContainerSheet({
  open,
  onOpenChange,
  defaultImage = "",
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  defaultImage?: string
}) {
  const { t } = useTranslation()
  const [optionsOpen, setOptionsOpen] = useState(true)
  const createMutation = useCreateContainer()
  const startMutation = useStartContainer()
  const busy = createMutation.isPending || startMutation.isPending

  const form = useForm<RunContainerForm>({
    defaultValues: {
      image: defaultImage,
      name: "",
      ports: [{ hostPort: "", containerPort: "" }],
      volumes: [{ hostPath: "", containerPath: "" }],
      envVars: [{ key: "", value: "" }],
    },
  })

  // Sync image when defaultImage changes (e.g. different image row)
  useEffect(() => {
    form.setValue("image", defaultImage)
  }, [defaultImage]) // eslint-disable-line react-hooks/exhaustive-deps

  const ports = useFieldArray({ control: form.control, name: "ports" })
  const volumes = useFieldArray({ control: form.control, name: "volumes" })
  const envVars = useFieldArray({ control: form.control, name: "envVars" })

  const handleClose = (v: boolean) => {
    if (!v)
      form.reset({
        image: defaultImage,
        name: "",
        ports: [{ hostPort: "", containerPort: "" }],
        volumes: [{ hostPath: "", containerPath: "" }],
        envVars: [{ key: "", value: "" }],
      })
    onOpenChange(v)
  }

  const onSubmit = form.handleSubmit((values) => {
    if (!values.image.trim()) {
      form.setError("image", { message: t("validation.required") })
      return
    }

    // Convert form arrays → API shape
    const port_bindings: Record<string, string> = {}
    for (const p of values.ports) {
      if (p.containerPort.trim()) {
        port_bindings[p.containerPort.trim()] = p.hostPort.trim() || "0"
      }
    }

    const vols = values.volumes
      .filter((v) => v.hostPath.trim() && v.containerPath.trim())
      .map((v) => `${v.hostPath.trim()}:${v.containerPath.trim()}`)

    const env: Record<string, string> = {}
    for (const e of values.envVars) {
      if (e.key.trim()) env[e.key.trim()] = e.value
    }

    const payload: CreateContainerModel = {
      image: values.image.trim(),
      name: values.name.trim() || undefined,
      port_bindings: Object.keys(port_bindings).length
        ? port_bindings
        : undefined,
      volumes: vols.length ? vols : undefined,
      env: Object.keys(env).length ? env : undefined,
    }

    createMutation.mutate(payload, {
      onSuccess: (container) => {
        startMutation.mutate(container.id, {
          onSuccess: () => {
            toast.success(t("docker.container.created"))
            handleClose(false)
          },
        })
      },
    })
  })

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="flex flex-col overflow-y-auto sm:max-w-lg">
        <SheetHeader className="border-b pb-4">
          <SheetTitle>{t("docker.container.runTitle")}</SheetTitle>
          {defaultImage && (
            <p className="font-mono text-sm text-muted-foreground">
              {defaultImage}
            </p>
          )}
        </SheetHeader>

        <form onSubmit={onSubmit} className="flex flex-1 flex-col gap-0">
          <div className="px-4">
            {/* Image input — only shown when not pre-set */}
            {!defaultImage && (
              <div className="flex flex-col gap-1.5 border-b py-4">
                <Label>{t("docker.container.imageLabel")}</Label>
                <Input placeholder="nginx:latest" {...form.register("image")} />
                {form.formState.errors.image && (
                  <p className="text-xs text-destructive">
                    {form.formState.errors.image.message}
                  </p>
                )}
              </div>
            )}

            {/* Optional settings collapsible */}
            <button
              type="button"
              onClick={() => setOptionsOpen((v) => !v)}
              className="flex w-full items-center justify-between border-b py-4 text-left text-sm font-medium"
            >
              <span>{t("docker.container.optionalSettings")}</span>
              {optionsOpen ? (
                <ChevronDownIcon className="size-4 text-muted-foreground" />
              ) : (
                <ChevronRightIcon className="size-4 text-muted-foreground" />
              )}
            </button>

            {optionsOpen && (
              <div className="flex flex-col gap-5 py-4">
                {/* Container name */}
                <div className="flex flex-col gap-1.5">
                  <Label>{t("docker.container.nameLabel")}</Label>
                  <Input
                    placeholder={t("docker.container.namePlaceholder")}
                    {...form.register("name", {
                      pattern: {
                        value: /^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/,
                        message: t("docker.container.nameInvalid"),
                      },
                    })}
                  />
                  {form.formState.errors.name ? (
                    <p className="text-xs text-destructive">
                      {form.formState.errors.name.message}
                    </p>
                  ) : (
                    <p className="text-xs text-muted-foreground">
                      {t("docker.container.nameHint")}
                    </p>
                  )}
                </div>

                {/* Ports */}
                <div className="flex flex-col gap-2">
                  <Label>{t("docker.container.portsLabel")}</Label>
                  <p className="-mt-1 text-xs text-muted-foreground">
                    {t("docker.container.portsHint")}
                  </p>
                  {ports.fields.map((field, i) => (
                    <div key={field.id} className="flex items-center gap-2">
                      <Input
                        className="flex-1"
                        placeholder={t("docker.container.hostPort")}
                        {...form.register(`ports.${i}.hostPort`)}
                      />
                      <span className="shrink-0 text-sm text-muted-foreground">
                        :
                      </span>
                      <Input
                        className="flex-1"
                        placeholder="80"
                        {...form.register(`ports.${i}.containerPort`)}
                      />
                      <span className="shrink-0 text-xs text-muted-foreground">
                        /tcp
                      </span>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-8 shrink-0"
                        onClick={() =>
                          ports.fields.length > 1
                            ? ports.remove(i)
                            : ports.update(i, {
                                hostPort: "",
                                containerPort: "",
                              })
                        }
                      >
                        <MinusIcon className="size-3.5" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="self-start"
                    onClick={() =>
                      ports.append({ hostPort: "", containerPort: "" })
                    }
                  >
                    <PlusIcon className="mr-1 size-3.5" />
                    {t("docker.container.addPort")}
                  </Button>
                </div>

                {/* Volumes */}
                <div className="flex flex-col gap-2">
                  <Label>{t("docker.container.volumesLabel")}</Label>
                  {volumes.fields.map((field, i) => (
                    <div key={field.id} className="flex items-center gap-2">
                      <Input
                        className="flex-1"
                        placeholder={t("docker.container.hostPath")}
                        {...form.register(`volumes.${i}.hostPath`)}
                      />
                      <Input
                        className="flex-1"
                        placeholder={t("docker.container.containerPath")}
                        {...form.register(`volumes.${i}.containerPath`)}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-8 shrink-0"
                        onClick={() =>
                          volumes.fields.length > 1
                            ? volumes.remove(i)
                            : volumes.update(i, {
                                hostPath: "",
                                containerPath: "",
                              })
                        }
                      >
                        <MinusIcon className="size-3.5" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="self-start"
                    onClick={() =>
                      volumes.append({ hostPath: "", containerPath: "" })
                    }
                  >
                    <PlusIcon className="mr-1 size-3.5" />
                    {t("docker.container.addVolume")}
                  </Button>
                </div>

                {/* Env vars */}
                <div className="flex flex-col gap-2">
                  <Label>{t("docker.container.envLabel")}</Label>
                  {envVars.fields.map((field, i) => (
                    <div key={field.id} className="flex items-center gap-2">
                      <Input
                        className="flex-1"
                        placeholder={t("docker.container.envKey")}
                        {...form.register(`envVars.${i}.key`)}
                      />
                      <Input
                        className="flex-1"
                        placeholder={t("docker.container.envValue")}
                        {...form.register(`envVars.${i}.value`)}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-8 shrink-0"
                        onClick={() =>
                          envVars.fields.length > 1
                            ? envVars.remove(i)
                            : envVars.update(i, { key: "", value: "" })
                        }
                      >
                        <MinusIcon className="size-3.5" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="self-start"
                    onClick={() => envVars.append({ key: "", value: "" })}
                  >
                    <PlusIcon className="mr-1 size-3.5" />
                    {t("docker.container.addEnv")}
                  </Button>
                </div>
              </div>
            )}
          </div>

          <SheetFooter className="mt-auto gap-2 border-t pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => handleClose(false)}
            >
              {t("docker.container.cancel")}
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? t("docker.actions.running") : t("docker.actions.run")}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ── Images tab ────────────────────────────────────────────────────────────────

function ImagesTab() {
  const { t } = useTranslation()
  const [showPull, setShowPull] = useState(false)
  const [runImage, setRunImage] = useState<string | null>(null)

  const { data: images, isLoading } = useGetImageList()
  const removeMutation = useRemoveImage()

  const handleRemove = (id: string) => {
    removeMutation.mutate(
      { id, force: false },
      {
        onSuccess: () => toast.success(t("docker.image.removed")),
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
        <p className="text-sm text-muted-foreground">
          {t("docker.image.empty")}
        </p>
      ) : (
        <div className="overflow-hidden rounded-lg border">
          <ImageTableHeader />
          {(images ?? []).map((img) => (
            <ImageRow
              key={img.id}
              image={img}
              onRun={(tag) => setRunImage(tag)}
              onRemove={handleRemove}
              busy={removeMutation.isPending}
            />
          ))}
        </div>
      )}

      <PullImageSheet open={showPull} onOpenChange={setShowPull} />
      <RunContainerSheet
        open={runImage !== null}
        onOpenChange={(v) => {
          if (!v) setRunImage(null)
        }}
        defaultImage={runImage ?? ""}
      />
    </div>
  )
}

// ── Image table header ────────────────────────────────────────────────────────

const IMG_COLS =
  "grid grid-cols-[minmax(0,3fr)_minmax(0,1fr)_minmax(0,1fr)_8rem]"

function ImageTableHeader() {
  const { t } = useTranslation()
  return (
    <div
      className={`${IMG_COLS} gap-4 border-b px-4 py-2 text-xs font-medium text-muted-foreground`}
    >
      <span>{t("docker.image.col.tag")}</span>
      <span>{t("docker.image.col.id")}</span>
      <span>{t("docker.image.col.size")}</span>
      <span className="text-right">{t("docker.image.col.actions")}</span>
    </div>
  )
}

function ImageRow({
  image,
  onRun,
  onRemove,
  busy,
}: {
  image: ImageModel
  onRun: (tag: string) => void
  onRemove: (id: string) => void
  busy: boolean
}) {
  const { t } = useTranslation()
  const tag = image.tags[0] ?? "<none>"

  return (
    <div
      className={`${IMG_COLS} items-center gap-4 border-b px-4 py-2.5 last:border-0 hover:bg-muted/30`}
    >
      <div className="flex min-w-0 items-center gap-2">
        <span className="truncate font-mono text-sm">{tag}</span>
        {image.in_use > 0 && (
          <Badge variant="secondary" className="shrink-0">
            {t("docker.image.inUse", { count: image.in_use })}
          </Badge>
        )}
      </div>
      <span className="font-mono text-xs text-muted-foreground">
        {image.short_id}
      </span>
      <span className="text-xs text-muted-foreground">{image.size}</span>
      <div className="flex justify-end gap-1">
        <Button
          variant="outline"
          size="sm"
          disabled={busy || tag === "<none>"}
          onClick={() => onRun(tag)}
        >
          <StartIcon className="size-3.5" />
        </Button>
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
  const queryClient = useQueryClient()
  const {
    headerLogs,
    layers,
    layerOrder,
    footerLogs,
    done,
    error,
    connected,
    pull,
    reset,
  } = usePullImageLogs()
  const bottomRef = useRef<HTMLDivElement>(null)

  const form = useForm<PullImageModel>({
    resolver: zodResolver(pullImageSchema),
    defaultValues: { reference: "" },
  })

  // Auto-scroll as logs arrive
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [footerLogs, layerOrder])

  // Refresh image list when pull completes successfully
  useEffect(() => {
    if (done && !error) {
      queryClient.invalidateQueries({ queryKey: CONTAINER_QUERY_KEYS.images() })
      toast.success(t("docker.image.pulled"))
    }
  }, [done, error]) // eslint-disable-line react-hooks/exhaustive-deps

  const onSubmit = form.handleSubmit(({ reference }) => pull(reference))

  const handleClose = (v: boolean) => {
    if (!v) {
      reset()
      form.reset()
    }
    onOpenChange(v)
  }

  const isPulling = connected

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="flex flex-col">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            {t("docker.image.pullTitle")}
            {isPulling && (
              <span className="animate-pulse text-xs text-muted-foreground">
                {t("docker.actions.pulling")}
              </span>
            )}
          </SheetTitle>
        </SheetHeader>

        <form onSubmit={onSubmit} className="mt-4 flex gap-2 px-4">
          <Input
            placeholder="nginx:latest"
            className="flex-1"
            disabled={isPulling}
            {...form.register("reference")}
          />
          <Button type="submit" disabled={isPulling}>
            {isPulling ? t("docker.actions.pulling") : t("docker.actions.pull")}
          </Button>
        </form>
        {form.formState.errors.reference && (
          <p className="-mt-2 text-xs text-destructive">
            {form.formState.errors.reference.message}
          </p>
        )}

        {(headerLogs ||
          layerOrder.length > 0 ||
          footerLogs ||
          isPulling ||
          error) && (
          <div className="mt-3 min-h-[200px] flex-1 overflow-auto rounded-md bg-muted p-3 font-mono text-xs">
            {/* "Pulling from library/nginx" and similar header lines */}
            {headerLogs && (
              <pre className="mb-2 leading-relaxed break-all whitespace-pre-wrap">
                {headerLogs}
              </pre>
            )}

            {/* Per-layer progress rows */}
            {layerOrder.length > 0 && (
              <div
                className="mb-2 grid gap-0.5"
                style={{ gridTemplateColumns: "5rem 8rem 1fr" }}
              >
                {layerOrder.map((id) => {
                  const layer = layers.get(id)
                  if (!layer) return null
                  return (
                    <div key={id} className="contents">
                      <span className="truncate text-muted-foreground">
                        {id}
                      </span>
                      <span className="truncate">{layer.status}</span>
                      <span className="truncate text-cyan-400">
                        {layer.progress}
                      </span>
                    </div>
                  )
                })}
              </div>
            )}

            {/* "Digest: sha256:..." and "Status: Downloaded newer image for ..." */}
            {footerLogs && (
              <pre className="leading-relaxed break-all whitespace-pre-wrap">
                {footerLogs}
              </pre>
            )}

            {isPulling && !layerOrder.length && !headerLogs && (
              <span className="animate-pulse">▌</span>
            )}
            {error && (
              <span className="mt-1 block text-destructive">
                [error] {error}
              </span>
            )}
            <div ref={bottomRef} />
          </div>
        )}
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
            <TabsTrigger value="containers">
              {t("docker.tabs.containers")}
            </TabsTrigger>
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
