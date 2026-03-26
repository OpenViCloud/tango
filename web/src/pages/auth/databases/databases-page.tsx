import { useEffect, useRef, useState } from "react"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"
import { PlusIcon, MinusIcon } from "lucide-react"
import { useQueryClient } from "@tanstack/react-query"

import type { ContainerModel } from "@/@types/models"
import {
  useGetImageList,
  useCreateContainer,
  useStartContainer,
  CONTAINER_QUERY_KEYS,
} from "@/hooks/api/use-container"
import { usePullImageLogs } from "@/hooks/api/use-pull-image-logs"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetFooter } from "@/components/ui/sheet"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Badge } from "@/components/ui/badge"
import { appIcons } from "@/lib/icons"

const DbIcon = appIcons.databases

// ── Database presets ──────────────────────────────────────────────────────────

type EnvPreset = { key: string; value: string }

type DbPreset = {
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
}

const DB_PRESETS: DbPreset[] = [
  {
    id: "postgres",
    name: "PostgreSQL",
    image: "postgres",
    description: "Advanced open source relational database with full SQL compliance.",
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
      { key: "MYSQL_USER", value: "" },
      { key: "MYSQL_PASSWORD", value: "" },
    ],
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
  },
  {
    id: "mariadb",
    name: "MariaDB",
    image: "mariadb",
    description: "Community-developed fork of MySQL with enterprise features.",
    color: "#003545",
    abbr: "MB",
    tags: ["latest", "11.4", "11.2", "10.11", "10.6"],
    port: { host: "3306", container: "3306" },
    dataPath: "/var/lib/mysql",
    env: [
      { key: "MARIADB_ROOT_PASSWORD", value: "root" },
      { key: "MARIADB_DATABASE", value: "mydb" },
    ],
  },
  {
    id: "elasticsearch",
    name: "Elasticsearch",
    image: "elasticsearch",
    description: "Distributed search and analytics engine for all types of data.",
    color: "#F04E98",
    abbr: "ES",
    tags: ["8.15.3", "8.14.3", "8.13.4", "7.17.24"],
    port: { host: "9200", container: "9200" },
    dataPath: "/usr/share/elasticsearch/data",
    env: [
      { key: "discovery.type", value: "single-node" },
      { key: "xpack.security.enabled", value: "false" },
    ],
  },
  {
    id: "cassandra",
    name: "Cassandra",
    image: "cassandra",
    description: "Highly scalable distributed NoSQL database.",
    color: "#1287B1",
    abbr: "CS",
    tags: ["latest", "5.0", "4.1", "4.0"],
    port: { host: "9042", container: "9042" },
    dataPath: "/var/lib/cassandra",
    env: [],
  },
  {
    id: "rabbitmq",
    name: "RabbitMQ",
    image: "rabbitmq",
    description: "Reliable and mature messaging and streaming broker.",
    color: "#FF6600",
    abbr: "RQ",
    tags: ["management", "latest", "4.0-management", "3.13-management"],
    port: { host: "5672", container: "5672" },
    dataPath: "/var/lib/rabbitmq",
    env: [
      { key: "RABBITMQ_DEFAULT_USER", value: "admin" },
      { key: "RABBITMQ_DEFAULT_PASS", value: "admin" },
    ],
  },
]

// ── Database card ─────────────────────────────────────────────────────────────

function DatabaseCard({
  preset,
  onDeploy,
}: {
  preset: DbPreset
  onDeploy: (preset: DbPreset) => void
}) {
  const { t } = useTranslation()
  const { data: images } = useGetImageList()
  const isLocal = images?.some((img) =>
    img.tags.some((tag) => tag.startsWith(`${preset.image}:`))
  ) ?? false

  return (
    <div className="flex flex-col gap-3 rounded-xl border bg-card p-4 hover:shadow-sm transition-shadow">
      <div className="flex items-start gap-3">
        <div
          className="flex size-10 shrink-0 items-center justify-center rounded-lg text-white text-xs font-bold"
          style={{ backgroundColor: preset.color }}
        >
          {preset.abbr}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium text-sm">{preset.name}</span>
            {isLocal && (
              <Badge variant="secondary" className="text-xs h-4 px-1.5">
                {t("databases.card.local")}
              </Badge>
            )}
          </div>
          <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">
            {preset.description}
          </p>
        </div>
      </div>

      <div className="flex flex-wrap gap-1">
        {preset.tags.slice(0, 4).map((tag) => (
          <Badge key={tag} variant="outline" className="text-xs font-mono px-1.5 h-5">
            {tag}
          </Badge>
        ))}
      </div>

      <Button
        size="sm"
        className="w-full mt-auto"
        onClick={() => onDeploy(preset)}
      >
        {t("databases.card.deploy")}
      </Button>
    </div>
  )
}

// ── Deploy database sheet ─────────────────────────────────────────────────────

type DeployPhase = "idle" | "pulling" | "creating" | "done"
type EnvEntry = { key: string; value: string }

function DeployDatabaseSheet({
  preset,
  open,
  onOpenChange,
}: {
  preset: DbPreset | null
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data: images } = useGetImageList()
  const createMutation = useCreateContainer()
  const startMutation = useStartContainer()
  const { headerLogs, layers, layerOrder, footerLogs, done: pullDone, error: pullError, connected: pulling, pull, reset: resetPull } = usePullImageLogs()
  const bottomRef = useRef<HTMLDivElement>(null)

  const [phase, setPhase] = useState<DeployPhase>("idle")
  const [tag, setTag] = useState("")
  const [name, setName] = useState("")
  const [nameError, setNameError] = useState("")
  const [hostPort, setHostPort] = useState("")
  const [hostDataPath, setHostDataPath] = useState("")
  const [envEntries, setEnvEntries] = useState<EnvEntry[]>([])

  // Reset form when preset changes
  useEffect(() => {
    if (preset) {
      setTag(preset.tags[0] ?? "latest")
      setName("")
      setNameError("")
      setHostPort(preset.port.host)
      setHostDataPath("")
      setEnvEntries(preset.env.map((e) => ({ key: e.key, value: e.value })))
      setPhase("idle")
      resetPull()
    }
  }, [preset]) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-scroll pull logs
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [footerLogs, layerOrder.length])

  // When pull finishes → create + start
  useEffect(() => {
    if (phase !== "pulling") return
    if (pullError) {
      toast.error(pullError)
      setPhase("idle")
      return
    }
    if (pullDone) {
      setPhase("creating")
      doCreateAndStart()
    }
  }, [pullDone, pullError]) // eslint-disable-line react-hooks/exhaustive-deps

  const isImageLocal = () => {
    if (!preset) return false
    const ref = `${preset.image}:${tag}`
    return images?.some((img) => img.tags.includes(ref)) ?? false
  }

  const buildPayload = () => {
    if (!preset) return null
    const env: Record<string, string> = {}
    for (const e of envEntries) {
      if (e.key.trim()) env[e.key.trim()] = e.value
    }
    const volumes: string[] = []
    if (hostDataPath.trim()) {
      volumes.push(`${hostDataPath.trim()}:${preset.dataPath}`)
    }
    return {
      image: `${preset.image}:${tag}`,
      name: name.trim() || undefined,
      port_bindings: hostPort.trim()
        ? { [preset.port.container]: hostPort.trim() }
        : undefined,
      volumes: volumes.length ? volumes : undefined,
      env: Object.keys(env).length ? env : undefined,
    }
  }

  const doCreateAndStart = () => {
    const payload = buildPayload()
    if (!payload) return
    createMutation.mutate(payload, {
      onSuccess: (container: ContainerModel) => {
        startMutation.mutate(container.id, {
          onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: CONTAINER_QUERY_KEYS.containers(false) })
            toast.success(t("databases.deploy.success", { name: preset?.name }))
            setPhase("done")
            onOpenChange(false)
          },
          onError: (err) => {
            toast.error(err.message)
            setPhase("idle")
          },
        })
      },
      onError: (err) => {
        toast.error(err.message)
        setPhase("idle")
      },
    })
  }

  const handleDeploy = () => {
    if (!preset) return
    if (name.trim() && !/^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/.test(name.trim())) {
      setNameError(t("docker.container.nameInvalid"))
      return
    }
    setNameError("")
    if (isImageLocal()) {
      setPhase("creating")
      doCreateAndStart()
    } else {
      setPhase("pulling")
      pull(`${preset.image}:${tag}`)
    }
  }

  const handleClose = (v: boolean) => {
    if (!v && phase === "pulling") {
      resetPull()
    }
    if (!v) setPhase("idle")
    onOpenChange(v)
  }

  const busy = phase === "pulling" || phase === "creating"

  if (!preset) return null

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="flex flex-col overflow-y-auto sm:max-w-lg">
        <SheetHeader className="border-b pb-4">
          <div className="flex items-center gap-3">
            <div
              className="flex size-9 shrink-0 items-center justify-center rounded-lg text-white text-xs font-bold"
              style={{ backgroundColor: preset.color }}
            >
              {preset.abbr}
            </div>
            <div>
              <SheetTitle>{t("databases.deploy.title", { name: preset.name })}</SheetTitle>
              <p className="text-xs text-muted-foreground mt-0.5">{preset.description}</p>
            </div>
          </div>
        </SheetHeader>

        <div className="flex flex-col gap-5 py-4 flex-1">
          {/* Version / tag */}
          <div className="flex flex-col gap-1.5">
            <Label>{t("databases.deploy.versionLabel")}</Label>
            <Select value={tag} onValueChange={setTag} disabled={busy}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {preset.tags.map((t) => (
                  <SelectItem key={t} value={t}>
                    {t}
                    {isImageLocal() && tag === t && (
                      <span className="ml-2 text-xs text-muted-foreground">
                        (local)
                      </span>
                    )}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Container name */}
          <div className="flex flex-col gap-1.5">
            <Label>{t("docker.container.nameLabel")}</Label>
            <Input
              placeholder={`${preset.id} (${t("databases.deploy.optional")})`}
              value={name}
              onChange={(e) => { setName(e.target.value); setNameError("") }}
              disabled={busy}
            />
            {nameError
              ? <p className="text-xs text-destructive">{nameError}</p>
              : <p className="text-xs text-muted-foreground">{t("docker.container.nameHint")}</p>
            }
          </div>

          {/* Port */}
          <div className="flex flex-col gap-1.5">
            <Label>{t("docker.container.portsLabel")}</Label>
            <div className="flex items-center gap-2">
              <Input
                className="flex-1"
                placeholder={t("docker.container.hostPort")}
                value={hostPort}
                onChange={(e) => setHostPort(e.target.value)}
                disabled={busy}
              />
              <span className="text-sm text-muted-foreground shrink-0">:</span>
              <div className="flex-1 rounded-lg border bg-muted/50 px-3 py-2 text-sm font-mono text-muted-foreground">
                {preset.port.container}/tcp
              </div>
            </div>
          </div>

          {/* Data volume */}
          <div className="flex flex-col gap-1.5">
            <Label>{t("databases.deploy.volumeLabel")}</Label>
            <div className="flex items-center gap-2">
              <Input
                className="flex-1"
                placeholder={t("docker.container.hostPath")}
                value={hostDataPath}
                onChange={(e) => setHostDataPath(e.target.value)}
                disabled={busy}
              />
              <span className="text-sm text-muted-foreground shrink-0">:</span>
              <div className="flex-1 rounded-lg border bg-muted/50 px-3 py-2 text-xs font-mono text-muted-foreground truncate">
                {preset.dataPath}
              </div>
            </div>
            <p className="text-xs text-muted-foreground">{t("databases.deploy.volumeHint")}</p>
          </div>

          {/* Environment variables */}
          {envEntries.length > 0 && (
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
                    onClick={() => setEnvEntries(envEntries.filter((_, j) => j !== i))}
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
                onClick={() => setEnvEntries([...envEntries, { key: "", value: "" }])}
              >
                <PlusIcon className="size-3.5 mr-1" />
                {t("docker.container.addEnv")}
              </Button>
            </div>
          )}

          {/* Pull progress */}
          {phase === "pulling" && (
            <div className="flex flex-col gap-1.5">
              <Label>{t("databases.deploy.pulling", { image: `${preset.image}:${tag}` })}</Label>
              <div className="rounded-md bg-muted p-3 text-xs font-mono min-h-[120px] max-h-[200px] overflow-auto">
                {headerLogs && (
                  <pre className="whitespace-pre-wrap break-all leading-relaxed mb-2">{headerLogs}</pre>
                )}
                {layerOrder.length > 0 && (
                  <div className="grid gap-0.5 mb-2" style={{ gridTemplateColumns: "5rem 8rem 1fr" }}>
                    {layerOrder.map((id) => {
                      const layer = layers.get(id)
                      if (!layer) return null
                      return (
                        <div key={id} className="contents">
                          <span className="text-muted-foreground truncate">{id}</span>
                          <span className="truncate">{layer.status}</span>
                          <span className="text-cyan-400 truncate">{layer.progress}</span>
                        </div>
                      )
                    })}
                  </div>
                )}
                {footerLogs && (
                  <pre className="whitespace-pre-wrap break-all leading-relaxed">{footerLogs}</pre>
                )}
                {pulling && !layerOrder.length && !headerLogs && (
                  <span className="animate-pulse">▌</span>
                )}
                <div ref={bottomRef} />
              </div>
            </div>
          )}
        </div>

        <SheetFooter className="border-t pt-4 gap-2">
          <Button variant="outline" disabled={busy} onClick={() => handleClose(false)}>
            {t("docker.container.cancel")}
          </Button>
          <Button disabled={busy} onClick={handleDeploy}>
            {phase === "pulling"
              ? t("databases.deploy.pulling_btn")
              : phase === "creating"
              ? t("databases.deploy.creating_btn")
              : t("databases.deploy.deployBtn")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export function DatabasesPage() {
  const { t } = useTranslation()
  const [selected, setSelected] = useState<DbPreset | null>(null)
  const [sheetOpen, setSheetOpen] = useState(false)

  const handleDeploy = (preset: DbPreset) => {
    setSelected(preset)
    setSheetOpen(true)
  }

  const handleSheetClose = (v: boolean) => {
    setSheetOpen(v)
    if (!v) setTimeout(() => setSelected(null), 300) // wait for sheet close animation
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<DbIcon />}
        title={t("databases.page.title")}
        description={t("databases.page.description")}
      />
      <SectionCard>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {DB_PRESETS.map((preset) => (
            <DatabaseCard key={preset.id} preset={preset} onDeploy={handleDeploy} />
          ))}
        </div>
      </SectionCard>

      <DeployDatabaseSheet
        preset={selected}
        open={sheetOpen}
        onOpenChange={handleSheetClose}
      />
    </div>
  )
}
