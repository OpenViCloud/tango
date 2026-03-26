import { zodResolver } from "@hookform/resolvers/zod"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  ArrowLeftIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  FolderIcon,
  MinusIcon,
  PlusIcon,
} from "lucide-react"
import { useEffect, useRef, useState } from "react"
import { useForm } from "react-hook-form"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import type {
  CreateEnvironmentModel,
  EnvironmentModel,
  ResourceModel,
  ResourceRunModel,
} from "@/@types/models"
import type { CreateResourceModel } from "@/@types/models/project"
import { createEnvironmentSchema } from "@/@types/models/project"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import {
  PROJECT_QUERY_KEYS,
  useCreateEnvironment,
  useCreateResource,
  useDeleteEnvironment,
  useDeleteResource,
  useGetProject,
  useStartResource,
  useStopResource,
} from "@/hooks/api/use-project"
import { useResourceRunLogs } from "@/hooks/api/use-resource-run-logs"
import { actionIcons } from "@/lib/icons"
import { Route } from "@/routes/_auth/projects/$projectId"
import { useQueryClient } from "@tanstack/react-query"

const StartIcon = actionIcons.start
const StopIcon = actionIcons.stop
const DeleteIcon = actionIcons.delete
const CreateIcon = actionIcons.create
const EditIcon = actionIcons.edit

// ── DB presets (shared with databases page pattern) ────────────────────────────

type EnvPreset = { key: string; value: string }
type ResourcePreset = {
  id: string
  name: string
  image: string
  description: string
  color: string
  abbr: string
  tags: string[]
  port: { host: string; container: string }
  dataPath: string
  env: EnvPreset[]
  type: string
}

const RESOURCE_PRESETS: ResourcePreset[] = [
  {
    id: "postgres",
    name: "PostgreSQL",
    image: "postgres",
    description:
      "Advanced open source relational database with full SQL compliance.",
    color: "#336791",
    abbr: "PG",
    tags: ["latest", "17", "16", "15", "14", "13"],
    port: { host: "5432", container: "5432" },
    dataPath: "/var/lib/postgresql/data",
    env: [
      { key: "POSTGRES_PASSWORD", value: "postgres" },
      { key: "POSTGRES_USER", value: "postgres" },
      { key: "POSTGRES_DB", value: "postgres" },
    ],
    type: "db",
  },
  {
    id: "mysql",
    name: "MySQL",
    image: "mysql",
    description: "The world's most popular open source relational database.",
    color: "#4479A1",
    abbr: "MY",
    tags: ["latest", "9.0", "8.4", "8.0", "5.7"],
    port: { host: "3306", container: "3306" },
    dataPath: "/var/lib/mysql",
    env: [
      { key: "MYSQL_ROOT_PASSWORD", value: "root" },
      { key: "MYSQL_DATABASE", value: "mydb" },
    ],
    type: "db",
  },
  {
    id: "redis",
    name: "Redis",
    image: "redis",
    description: "In-memory data structure store, cache and message broker.",
    color: "#DC382D",
    abbr: "RD",
    tags: ["latest", "7.4", "7.2", "7.0", "6.2"],
    port: { host: "6379", container: "6379" },
    dataPath: "/data",
    env: [],
    type: "db",
  },
  {
    id: "mongo",
    name: "MongoDB",
    image: "mongo",
    description: "Document-oriented NoSQL database for modern applications.",
    color: "#00874A",
    abbr: "MG",
    tags: ["latest", "8.0", "7.0", "6.0", "5.0"],
    port: { host: "27017", container: "27017" },
    dataPath: "/data/db",
    env: [
      { key: "MONGO_INITDB_ROOT_USERNAME", value: "root" },
      { key: "MONGO_INITDB_ROOT_PASSWORD", value: "root" },
    ],
    type: "db",
  },
  {
    id: "rabbitmq",
    name: "RabbitMQ",
    image: "rabbitmq",
    description: "Reliable and mature messaging and streaming broker.",
    color: "#FF6600",
    abbr: "RQ",
    tags: ["management", "latest", "4.0-management"],
    port: { host: "5672", container: "5672" },
    dataPath: "/var/lib/rabbitmq",
    env: [
      { key: "RABBITMQ_DEFAULT_USER", value: "admin" },
      { key: "RABBITMQ_DEFAULT_PASS", value: "admin" },
    ],
    type: "service",
  },
  {
    id: "nginx",
    name: "Nginx",
    image: "nginx",
    description: "High-performance HTTP server and reverse proxy.",
    color: "#009639",
    abbr: "NG",
    tags: ["latest", "1.27", "1.26", "alpine"],
    port: { host: "8080", container: "80" },
    dataPath: "/usr/share/nginx/html",
    env: [],
    type: "app",
  },
]

// ── Status dot ────────────────────────────────────────────────────────────────

const STATUS_DOT: Record<string, string> = {
  running: "bg-green-500",
  stopped: "bg-muted-foreground/40",
  error: "bg-destructive",
  creating: "bg-yellow-500",
  pulling: "bg-yellow-500",
}

function StatusDot({ status }: { status: string }) {
  return (
    <span
      className={`inline-block size-2 shrink-0 rounded-full ${STATUS_DOT[status] ?? "bg-muted-foreground/40"}`}
    />
  )
}

// ── Resource card ─────────────────────────────────────────────────────────────

function ResourceCard({
  resource,
  onStart,
  onStop,
  onDelete,
  busy,
}: {
  resource: ResourceModel
  onStart: (resource: ResourceModel) => void
  onStop: (id: string) => void
  onDelete: (id: string) => void
  busy: boolean
}) {
  const { t } = useTranslation()
  const isRunning = resource.status === "running"
  const hasContainer = !!resource.container_id
  const canStart = !isRunning
  const canStop = isRunning && hasContainer

  const portSummary = resource.ports
    .map((p) => `${p.host_port}→${p.internal_port}`)
    .join(", ")

  // Derive abbr from name or image
  const abbr = resource.name.slice(0, 2).toUpperCase()
  const typeColor: Record<string, string> = {
    db: "#336791",
    app: "#009639",
    service: "#FF6600",
  }
  const color = typeColor[resource.type] ?? "#6b7280"

  return (
    <div className="flex flex-col gap-3 rounded-xl border bg-card p-4">
      <div className="flex items-start gap-3">
        <div
          className="flex size-9 shrink-0 items-center justify-center rounded-lg text-xs font-bold text-white"
          style={{ backgroundColor: color }}
        >
          {abbr}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <StatusDot status={resource.status} />
            <span className="truncate text-sm font-medium">
              {resource.name}
            </span>
          </div>
          <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
            {resource.image}:{resource.tag}
          </p>
        </div>
        <Badge variant="outline" className="shrink-0 text-xs">
          {resource.type}
        </Badge>
      </div>

      {portSummary && (
        <p className="font-mono text-xs text-muted-foreground">{portSummary}</p>
      )}

      <div className="mt-auto flex items-center gap-1.5">
        <Button asChild type="button" variant="ghost" size="sm" disabled={busy}>
          <Link
            to="/resources/$resourceId"
            params={{ resourceId: resource.id }}
          >
            <EditIcon className="size-3.5" />
          </Link>
        </Button>
        {canStop ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={() => onStop(resource.id)}
            className="flex-1"
          >
            <StopIcon className="mr-1 size-3.5" />
            {t("projects.resource.stop")}
          </Button>
        ) : (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={busy || !canStart}
            onClick={() => onStart(resource)}
            className="flex-1"
          >
            <StartIcon className="mr-1 size-3.5" />
            {t("projects.resource.start")}
          </Button>
        )}
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={busy}
          onClick={() => onDelete(resource.id)}
          className="text-destructive hover:text-destructive"
        >
          <DeleteIcon className="size-3.5" />
        </Button>
      </div>
    </div>
  )
}

// ── Deploy resource sheet ─────────────────────────────────────────────────────

type DeployPhase = "idle" | "preset" | "config" | "creating" | "done"
type EnvEntry = { key: string; value: string }
type PortEntry = { host: string; container: string }

function DeployResourceSheet({
  envId,
  projectId,
  open,
  onOpenChange,
}: {
  envId: string
  projectId: string
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const createMutation = useCreateResource(envId, projectId)

  const [phase, setPhase] = useState<DeployPhase>("preset")
  const [selectedPreset, setSelectedPreset] = useState<ResourcePreset | null>(
    null
  )
  const [customMode, setCustomMode] = useState(false)

  // Config form state
  const [name, setName] = useState("")
  const [nameError, setNameError] = useState("")
  const [tag, setTag] = useState("latest")
  const [customImage, setCustomImage] = useState("")
  const [resourceType, setResourceType] = useState("db")
  const [ports, setPorts] = useState<PortEntry[]>([{ host: "", container: "" }])
  const [envEntries, setEnvEntries] = useState<EnvEntry[]>([
    { key: "", value: "" },
  ])

  const resetForm = () => {
    setPhase("preset")
    setSelectedPreset(null)
    setCustomMode(false)
    setName("")
    setNameError("")
    setTag("latest")
    setCustomImage("")
    setResourceType("db")
    setPorts([{ host: "", container: "" }])
    setEnvEntries([{ key: "", value: "" }])
  }

  const handleClose = (v: boolean) => {
    if (!v) {
      resetForm()
    }
    onOpenChange(v)
  }

  const selectPreset = (preset: ResourcePreset) => {
    setSelectedPreset(preset)
    setCustomMode(false)
    setTag(preset.tags[0] ?? "latest")
    setName(preset.id)
    setResourceType(preset.type)
    setPorts([{ host: preset.port.host, container: preset.port.container }])
    setEnvEntries(
      preset.env.length > 0
        ? preset.env.map((e) => ({ key: e.key, value: e.value }))
        : [{ key: "", value: "" }]
    )
    setPhase("config")
  }

  const selectCustom = () => {
    setSelectedPreset(null)
    setCustomMode(true)
    setTag("latest")
    setName("")
    setResourceType("app")
    setPorts([{ host: "", container: "" }])
    setEnvEntries([{ key: "", value: "" }])
    setPhase("config")
  }

  const buildPayload = (): CreateResourceModel => {
    const envVars = envEntries
      .filter((e) => e.key.trim())
      .map((e) => ({ key: e.key.trim(), value: e.value, is_secret: false }))

    const portList = ports
      .filter((p) => p.container.trim())
      .map((p) => ({
        host_port: parseInt(p.host.trim() || "0", 10),
        internal_port: parseInt(p.container.trim(), 10),
        proto: "tcp",
        label: "",
      }))

    const image = customMode
      ? customImage.trim()
      : (selectedPreset?.image ?? "")

    return {
      name: name.trim(),
      type: resourceType,
      image,
      tag,
      ports: portList,
      env_vars: envVars,
    }
  }

  const doCreate = () => {
    const payload = buildPayload()
    createMutation.mutate(payload, {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: PROJECT_QUERY_KEYS.project(projectId),
        })
        toast.success(t("projects.resource.created"))
        setPhase("done")
        handleClose(false)
      },
      onError: (err) => {
        toast.error(err.message)
        setPhase("config")
      },
    })
  }

  const handleDeploy = () => {
    if (!name.trim()) {
      setNameError(t("validation.required") || "Required")
      return
    }
    setNameError("")
    const image = customMode
      ? customImage.trim()
      : (selectedPreset?.image ?? "")
    if (!image) {
      toast.error("Image is required")
      return
    }
    setPhase("creating")
    doCreate()
  }

  const busy = phase === "creating"

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="flex flex-col overflow-y-auto sm:max-w-lg">
        <SheetHeader className="border-b pb-4">
          <div className="flex items-center gap-2">
            {phase === "config" && (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="-ml-1 size-7"
                onClick={() => setPhase("preset")}
                disabled={busy}
              >
                <ArrowLeftIcon className="size-4" />
              </Button>
            )}
            <SheetTitle>
              {phase === "preset"
                ? t("projects.deployResource")
                : selectedPreset
                  ? selectedPreset.name
                  : t("projects.deployResource")}
            </SheetTitle>
          </div>
        </SheetHeader>

        {/* Preset picker */}
        {phase === "preset" && (
          <div className="flex flex-1 flex-col gap-4 py-4">
            <div className="grid grid-cols-2 gap-3">
              {RESOURCE_PRESETS.map((preset) => (
                <button
                  key={preset.id}
                  type="button"
                  onClick={() => selectPreset(preset)}
                  className="flex flex-col gap-2 rounded-xl border bg-card p-3 text-left transition-shadow hover:border-primary/40 hover:shadow-sm"
                >
                  <div className="flex items-center gap-2">
                    <div
                      className="flex size-8 shrink-0 items-center justify-center rounded-lg text-xs font-bold text-white"
                      style={{ backgroundColor: preset.color }}
                    >
                      {preset.abbr}
                    </div>
                    <div>
                      <p className="text-sm font-medium">{preset.name}</p>
                      <p className="text-xs text-muted-foreground">
                        {preset.type}
                      </p>
                    </div>
                  </div>
                  <p className="line-clamp-2 text-xs text-muted-foreground">
                    {preset.description}
                  </p>
                </button>
              ))}
              <button
                type="button"
                onClick={selectCustom}
                className="flex flex-col gap-2 rounded-xl border border-dashed bg-card p-3 text-left transition-shadow hover:border-primary/40 hover:shadow-sm"
              >
                <div className="flex items-center gap-2">
                  <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <PlusIcon className="size-4" />
                  </div>
                  <p className="text-sm font-medium">Custom</p>
                </div>
                <p className="text-xs text-muted-foreground">
                  Use any Docker image
                </p>
              </button>
            </div>
          </div>
        )}

        {/* Config form */}
        {phase === "config" || phase === "creating" ? (
          <div className="flex flex-1 flex-col gap-5 py-4">
            {/* Custom image input */}
            {customMode && (
              <div className="flex flex-col gap-1.5">
                <Label>Image</Label>
                <Input
                  placeholder="nginx"
                  value={customImage}
                  onChange={(e) => setCustomImage(e.target.value)}
                  disabled={busy}
                />
              </div>
            )}

            {/* Resource type */}
            <div className="flex flex-col gap-1.5">
              <Label>Type</Label>
              <Select
                value={resourceType}
                onValueChange={setResourceType}
                disabled={busy}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="db">Database</SelectItem>
                  <SelectItem value="app">Application</SelectItem>
                  <SelectItem value="service">Service</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Tag / version */}
            {selectedPreset ? (
              <div className="flex flex-col gap-1.5">
                <Label>{t("databases.deploy.versionLabel")}</Label>
                <Select value={tag} onValueChange={setTag} disabled={busy}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {selectedPreset.tags.map((t) => (
                      <SelectItem key={t} value={t}>
                        {t}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : (
              <div className="flex flex-col gap-1.5">
                <Label>Tag</Label>
                <Input
                  placeholder="latest"
                  value={tag}
                  onChange={(e) => setTag(e.target.value)}
                  disabled={busy}
                />
              </div>
            )}

            {/* Name */}
            <div className="flex flex-col gap-1.5">
              <Label>{t("projects.nameLabel")}</Label>
              <Input
                placeholder={t("projects.namePlaceholder")}
                value={name}
                onChange={(e) => {
                  setName(e.target.value)
                  setNameError("")
                }}
                disabled={busy}
              />
              {nameError && (
                <p className="text-xs text-destructive">{nameError}</p>
              )}
            </div>

            {/* Ports */}
            <div className="flex flex-col gap-2">
              <Label>Ports</Label>
              {ports.map((port, i) => (
                <div key={i} className="flex items-center gap-2">
                  <Input
                    className="flex-1"
                    placeholder="Host port"
                    value={port.host}
                    onChange={(e) => {
                      const next = [...ports]
                      next[i] = { ...next[i], host: e.target.value }
                      setPorts(next)
                    }}
                    disabled={busy}
                  />
                  <span className="shrink-0 text-sm text-muted-foreground">
                    :
                  </span>
                  <Input
                    className="flex-1"
                    placeholder="Container port"
                    value={port.container}
                    onChange={(e) => {
                      const next = [...ports]
                      next[i] = { ...next[i], container: e.target.value }
                      setPorts(next)
                    }}
                    disabled={busy}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-8 shrink-0"
                    disabled={busy}
                    onClick={() =>
                      ports.length > 1
                        ? setPorts(ports.filter((_, j) => j !== i))
                        : setPorts([{ host: "", container: "" }])
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
                disabled={busy}
                onClick={() =>
                  setPorts([...ports, { host: "", container: "" }])
                }
              >
                <PlusIcon className="mr-1 size-3.5" />
                Add port
              </Button>
            </div>

            {/* Env vars */}
            <div className="flex flex-col gap-2">
              <Label>{t("docker.container.envLabel")}</Label>
              {envEntries.map((entry, i) => (
                <div key={i} className="flex items-center gap-2">
                  <Input
                    className="flex-1 font-mono text-xs"
                    placeholder={t("docker.container.envKey")}
                    value={entry.key}
                    onChange={(e) => {
                      const next = [...envEntries]
                      next[i] = { ...next[i], key: e.target.value }
                      setEnvEntries(next)
                    }}
                    disabled={busy}
                  />
                  <Input
                    className="flex-1 text-xs"
                    placeholder={t("docker.container.envValue")}
                    value={entry.value}
                    onChange={(e) => {
                      const next = [...envEntries]
                      next[i] = { ...next[i], value: e.target.value }
                      setEnvEntries(next)
                    }}
                    disabled={busy}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-8 shrink-0"
                    disabled={busy}
                    onClick={() =>
                      envEntries.length > 1
                        ? setEnvEntries(envEntries.filter((_, j) => j !== i))
                        : setEnvEntries([{ key: "", value: "" }])
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
                disabled={busy}
                onClick={() =>
                  setEnvEntries([...envEntries, { key: "", value: "" }])
                }
              >
                <PlusIcon className="mr-1 size-3.5" />
                {t("docker.container.addEnv")}
              </Button>
            </div>
          </div>
        ) : null}

        {(phase === "config" || phase === "creating") && (
          <SheetFooter className="gap-2 border-t pt-4">
            <Button
              variant="outline"
              disabled={busy}
              onClick={() => handleClose(false)}
            >
              {t("projects.cancel")}
            </Button>
            <Button disabled={busy} onClick={handleDeploy}>
              {phase === "creating"
                ? t("databases.deploy.creating_btn")
                : t("projects.addResource")}
            </Button>
          </SheetFooter>
        )}
      </SheetContent>
    </Sheet>
  )
}

// ── Add environment dialog ────────────────────────────────────────────────────

function AddEnvironmentDialog({
  projectId,
  open,
  onOpenChange,
}: {
  projectId: string
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { t } = useTranslation()
  const createMutation = useCreateEnvironment(projectId)

  const form = useForm<CreateEnvironmentModel>({
    resolver: zodResolver(createEnvironmentSchema),
    defaultValues: { name: "" },
  })

  const handleClose = (v: boolean) => {
    if (!v) form.reset()
    onOpenChange(v)
  }

  const onSubmit = form.handleSubmit((values) => {
    createMutation.mutate(values, {
      onSuccess: () => {
        toast.success("Environment added.")
        handleClose(false)
      },
      onError: (err) => toast.error(err.message),
    })
  })

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="sm:max-w-sm">
        <SheetHeader>
          <SheetTitle>{t("projects.addEnv")}</SheetTitle>
        </SheetHeader>
        <form onSubmit={onSubmit} className="mt-4 flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="env-name">{t("projects.envNameLabel")}</Label>
            <Input
              id="env-name"
              placeholder={t("projects.envNamePlaceholder")}
              {...form.register("name")}
            />
            {form.formState.errors.name && (
              <p className="text-xs text-destructive">
                {form.formState.errors.name.message}
              </p>
            )}
          </div>
          <SheetFooter className="gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={createMutation.isPending}
              onClick={() => handleClose(false)}
            >
              {t("projects.cancel")}
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending
                ? t("projects.creating")
                : t("projects.create")}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

function ResourceRunLogSheet({
  run,
  resourceName,
  onClose,
  onCompleted,
}: {
  run: ResourceRunModel | null
  resourceName: string | null
  onClose: () => void
  onCompleted: () => void
}) {
  const { t } = useTranslation()
  const { logs, status, connected } = useResourceRunLogs(run?.id ?? null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const notifiedStatusRef = useRef<string | null>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [logs])

  useEffect(() => {
    notifiedStatusRef.current = null
  }, [run?.id])

  useEffect(() => {
    if (!status) return
    if (notifiedStatusRef.current === status) return
    notifiedStatusRef.current = status
    if (status === "done") {
      toast.success(t("projects.resource.started"))
      onCompleted()
      return
    }
    if (status === "failed") {
      toast.error(run?.error_msg || t("projects.resource.runFailed"))
      onCompleted()
    }
  }, [onCompleted, run?.error_msg, status, t])

  const displayStatus = status ?? run?.status
  const isEmpty = !logs && !connected

  return (
    <Sheet open={Boolean(run)} onOpenChange={(v) => !v && onClose()}>
      <SheetContent className="flex w-full flex-col sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            {t("projects.resource.logTitle")}
            {displayStatus ? (
              <Badge variant="outline" className="font-normal">
                {displayStatus}
              </Badge>
            ) : null}
            {connected ? (
              <span className="animate-pulse text-xs text-muted-foreground">
                {t("projects.resource.streaming")}
              </span>
            ) : null}
          </SheetTitle>
          {resourceName ? (
            <p className="truncate font-mono text-xs text-muted-foreground">
              {resourceName}
            </p>
          ) : null}
        </SheetHeader>

        <div className="mt-4 flex-1 overflow-auto">
          {isEmpty ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-5/6" />
            </div>
          ) : (
            <pre className="min-h-[220px] rounded-md bg-muted p-4 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap">
              {logs || t("projects.resource.logEmpty")}
              {connected ? <span className="animate-pulse">▌</span> : null}
              <div ref={bottomRef} />
            </pre>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

// ── Environment section ───────────────────────────────────────────────────────

function EnvironmentSection({
  env,
  projectId,
  onDeleteEnv,
  deletingEnvId,
}: {
  env: EnvironmentModel
  projectId: string
  onDeleteEnv: (envId: string) => void
  deletingEnvId: string | null
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(true)
  const [deployOpen, setDeployOpen] = useState(false)
  const [activeRun, setActiveRun] = useState<ResourceRunModel | null>(null)
  const [activeRunResourceName, setActiveRunResourceName] = useState<
    string | null
  >(null)
  const startMutation = useStartResource()
  const stopMutation = useStopResource()
  const deleteMutation = useDeleteResource(projectId)

  const actionBusy =
    activeRun !== null ||
    startMutation.isPending ||
    stopMutation.isPending ||
    deleteMutation.isPending

  const handleStart = (resource: ResourceModel) => {
    startMutation.mutate(resource.id, {
      onSuccess: (run) => {
        setActiveRun(run)
        setActiveRunResourceName(resource.name)
      },
      onError: (err) => toast.error(err.message),
    })
  }

  const handleRunCompleted = () => {
    queryClient.invalidateQueries({
      queryKey: PROJECT_QUERY_KEYS.project(projectId),
    })
  }

  const handleStop = (id: string) => {
    stopMutation.mutate(id, {
      onSuccess: () => toast.success(t("projects.resource.stopped")),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleDelete = (id: string) => {
    deleteMutation.mutate(id, {
      onSuccess: () => toast.success(t("projects.resource.deleted")),
      onError: (err) => toast.error(err.message),
    })
  }

  const isDeleting = deletingEnvId === env.id

  return (
    <div className="overflow-hidden rounded-xl border">
      {/* Environment header */}
      <div className="flex items-center gap-2 border-b bg-muted/30 px-4 py-3">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex flex-1 items-center gap-2 text-left"
        >
          {open ? (
            <ChevronDownIcon className="size-4 shrink-0 text-muted-foreground" />
          ) : (
            <ChevronRightIcon className="size-4 shrink-0 text-muted-foreground" />
          )}
          <span className="text-sm font-medium">{env.name}</span>
          <span className="text-xs text-muted-foreground">
            {env.resources.length} resource
            {env.resources.length !== 1 ? "s" : ""}
          </span>
        </button>
        <div className="flex items-center gap-1.5">
          <Button
            size="sm"
            variant="outline"
            onClick={() => setDeployOpen(true)}
          >
            <CreateIcon className="mr-1 size-3.5" />
            {t("projects.addResource")}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            disabled={isDeleting}
            onClick={() => onDeleteEnv(env.id)}
            className="text-destructive hover:text-destructive"
          >
            <DeleteIcon className="size-3.5" />
          </Button>
        </div>
      </div>

      {/* Resources grid */}
      {open && (
        <div className="p-4">
          {env.resources.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              {t("projects.noResources")}
            </p>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {env.resources.map((resource) => (
                <ResourceCard
                  key={resource.id}
                  resource={resource}
                  onStart={handleStart}
                  onStop={handleStop}
                  onDelete={handleDelete}
                  busy={actionBusy}
                />
              ))}
            </div>
          )}
        </div>
      )}

      <DeployResourceSheet
        envId={env.id}
        projectId={projectId}
        open={deployOpen}
        onOpenChange={setDeployOpen}
      />
      <ResourceRunLogSheet
        run={activeRun}
        resourceName={activeRunResourceName}
        onClose={() => {
          setActiveRun(null)
          setActiveRunResourceName(null)
        }}
        onCompleted={handleRunCompleted}
      />
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export function ProjectDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { projectId } = Route.useParams()
  const [addEnvOpen, setAddEnvOpen] = useState(false)

  const { data: project, isLoading } = useGetProject(projectId)
  const deleteEnvMutation = useDeleteEnvironment(projectId)
  const [deletingEnvId, setDeletingEnvId] = useState<string | null>(null)

  const handleDeleteEnv = (envId: string) => {
    setDeletingEnvId(envId)
    deleteEnvMutation.mutate(envId, {
      onSuccess: () => {
        toast.success("Environment deleted.")
        setDeletingEnvId(null)
      },
      onError: (err) => {
        toast.error(err.message)
        setDeletingEnvId(null)
      },
    })
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Back + header */}
      <div className="flex flex-col gap-4">
        <Button
          variant="ghost"
          size="sm"
          className="-ml-2 self-start"
          onClick={() => navigate({ to: "/projects" })}
        >
          <ArrowLeftIcon className="mr-1 size-4" />
          {t("projects.page.title")}
        </Button>

        {isLoading ? (
          <Skeleton className="h-14 w-64 rounded-xl" />
        ) : project ? (
          <PageHeaderCard
            icon={<FolderIcon className="size-5" />}
            title={project.name}
            description={project.description}
            headerRight={
              <Button size="sm" onClick={() => setAddEnvOpen(true)}>
                <CreateIcon data-icon="inline-start" />
                {t("projects.addEnv")}
              </Button>
            }
          />
        ) : null}
      </div>

      {/* Environments */}
      {isLoading ? (
        <SectionCard>
          <div className="flex flex-col gap-3">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-32 w-full rounded-xl" />
            ))}
          </div>
        </SectionCard>
      ) : project ? (
        project.environments.length === 0 ? (
          <SectionCard>
            <p className="text-sm text-muted-foreground">
              {t("projects.empty")}
            </p>
          </SectionCard>
        ) : (
          <div className="flex flex-col gap-4">
            {project.environments.map((env) => (
              <EnvironmentSection
                key={env.id}
                env={env}
                projectId={projectId}
                onDeleteEnv={handleDeleteEnv}
                deletingEnvId={deletingEnvId}
              />
            ))}
          </div>
        )
      ) : null}

      <AddEnvironmentDialog
        projectId={projectId}
        open={addEnvOpen}
        onOpenChange={setAddEnvOpen}
      />
    </div>
  )
}
