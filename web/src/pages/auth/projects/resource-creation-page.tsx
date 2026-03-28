import {
  ArrowLeftIcon,
  GitBranchIcon,
  GithubIcon,
  MinusIcon,
  PlusIcon,
  SearchIcon,
  ServerIcon,
} from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

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
  PROJECT_QUERY_KEYS,
  useCheckRepo,
  useCreateResource,
  useCreateResourceFromGit,
} from "@/hooks/api/use-project"
import {
  useGetSourceBranches,
  useGetSourceList,
  useGetSourceRepos,
} from "@/hooks/api/use-source"
import { useNavigate } from "@tanstack/react-router"
import { useQueryClient } from "@tanstack/react-query"

// ── Types ─────────────────────────────────────────────────────────────────────

type EnvEntry = { key: string; value: string }
type PortEntry = { host: string; container: string }
type FlowType = "preset" | "docker" | "git" | "git-private"
type Phase = "picker" | "config" | "git" | "submitting"

type ResourcePreset = {
  id: string
  name: string
  image: string
  description: string
  color: string
  abbr: string
  tags: string[]
  ports: PortEntry[]
  env: { key: string; value: string }[]
  type: string
}

// ── Presets ───────────────────────────────────────────────────────────────────

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
    ports: [{ host: "5432", container: "5432" }],
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
    ports: [{ host: "3306", container: "3306" }],
    env: [
      { key: "MYSQL_ROOT_PASSWORD", value: "root" },
      { key: "MYSQL_DATABASE", value: "mydb" },
    ],
    type: "db",
  },
  {
    id: "sqlserver",
    name: "SQL Server",
    image: "mcr.microsoft.com/mssql/server",
    description:
      "Microsoft SQL Server — enterprise relational database engine.",
    color: "#CC2927",
    abbr: "MS",
    tags: ["latest", "2022-latest", "2019-latest"],
    ports: [{ host: "1433", container: "1433" }],
    env: [
      { key: "ACCEPT_EULA", value: "Y" },
      { key: "SA_PASSWORD", value: "SqlServer@123" },
      { key: "MSSQL_PID", value: "Developer" },
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
    ports: [{ host: "6379", container: "6379" }],
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
    ports: [{ host: "27017", container: "27017" }],
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
    ports: [
      { host: "5672", container: "5672" },
      { host: "15672", container: "15672" },
    ],
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
    ports: [{ host: "8080", container: "80" }],
    env: [],
    type: "service",
  },
  {
    id: "kafka",
    name: "Apache Kafka",
    image: "confluentinc/cp-kafka",
    description:
      "Distributed event streaming platform for high-performance data pipelines.",
    color: "#231F20",
    abbr: "KF",
    tags: ["latest", "7.6", "7.5", "7.4"],
    ports: [{ host: "9092", container: "9092" }],
    env: [
      { key: "KAFKA_BROKER_ID", value: "1" },
      { key: "KAFKA_ZOOKEEPER_CONNECT", value: "zookeeper:2181" },
      {
        key: "KAFKA_ADVERTISED_LISTENERS",
        value: "PLAINTEXT://localhost:9092",
      },
    ],
    type: "service",
  },
  {
    id: "minio",
    name: "MinIO",
    image: "minio/minio",
    description: "High-performance S3-compatible object storage server.",
    color: "#C72E49",
    abbr: "MN",
    tags: ["latest"],
    ports: [
      { host: "9000", container: "9000" },
      { host: "9001", container: "9001" },
    ],
    env: [
      { key: "MINIO_ROOT_USER", value: "minioadmin" },
      { key: "MINIO_ROOT_PASSWORD", value: "minioadmin" },
    ],
    type: "service",
  },
  {
    id: "grafana",
    name: "Grafana",
    image: "grafana/grafana",
    description:
      "Open source analytics & interactive visualization web application.",
    color: "#F46800",
    abbr: "GF",
    tags: ["latest", "11.0.0", "10.4.0"],
    ports: [{ host: "3000", container: "3000" }],
    env: [
      { key: "GF_SECURITY_ADMIN_USER", value: "admin" },
      { key: "GF_SECURITY_ADMIN_PASSWORD", value: "admin" },
    ],
    type: "service",
  },
  {
    id: "n8n",
    name: "n8n",
    image: "n8nio/n8n",
    description: "Workflow automation tool — connect anything to everything.",
    color: "#EA4B71",
    abbr: "N8",
    tags: ["latest", "1.44.1"],
    ports: [{ host: "5678", container: "5678" }],
    env: [{ key: "N8N_BASIC_AUTH_ACTIVE", value: "true" }],
    type: "service",
  },
  {
    id: "gitea",
    name: "Gitea",
    image: "gitea/gitea",
    description: "Lightweight self-hosted Git service.",
    color: "#609926",
    abbr: "GT",
    tags: ["latest", "1.22", "1.21"],
    ports: [
      { host: "3000", container: "3000" },
      { host: "222", container: "22" },
    ],
    env: [],
    type: "service",
  },
]

// ── Sub-components ────────────────────────────────────────────────────────────

function PresetCard({
  preset,
  onClick,
}: {
  preset: ResourcePreset
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex flex-col gap-2 rounded-xl border bg-card p-3 text-left transition-shadow hover:border-primary/40 hover:shadow-sm"
    >
      <div className="flex items-center gap-2">
        <div
          className="flex size-9 shrink-0 items-center justify-center rounded-lg text-xs font-bold text-white"
          style={{ backgroundColor: preset.color }}
        >
          {preset.abbr}
        </div>
        <div>
          <p className="text-sm font-semibold">{preset.name}</p>
          <p className="text-xs text-muted-foreground">
            {preset.image.split("/").pop()}
          </p>
        </div>
      </div>
      <p className="line-clamp-2 text-xs text-muted-foreground">
        {preset.description}
      </p>
    </button>
  )
}

function PortList({
  ports,
  onChange,
  disabled,
}: {
  ports: PortEntry[]
  onChange: (ports: PortEntry[]) => void
  disabled: boolean
}) {
  return (
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
              onChange(next)
            }}
            disabled={disabled}
          />
          <span className="shrink-0 text-sm text-muted-foreground">:</span>
          <Input
            className="flex-1"
            placeholder="Container port"
            value={port.container}
            onChange={(e) => {
              const next = [...ports]
              next[i] = { ...next[i], container: e.target.value }
              onChange(next)
            }}
            disabled={disabled}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-8 shrink-0"
            disabled={disabled}
            onClick={() =>
              ports.length > 1
                ? onChange(ports.filter((_, j) => j !== i))
                : onChange([{ host: "", container: "" }])
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
        disabled={disabled}
        onClick={() => onChange([...ports, { host: "", container: "" }])}
      >
        <PlusIcon className="mr-1 size-3.5" />
        Add port
      </Button>
    </div>
  )
}

function EnvList({
  entries,
  onChange,
  disabled,
}: {
  entries: EnvEntry[]
  onChange: (entries: EnvEntry[]) => void
  disabled: boolean
}) {
  return (
    <div className="flex flex-col gap-2">
      <Label>Environment Variables</Label>
      {entries.map((entry, i) => (
        <div key={i} className="flex items-center gap-2">
          <Input
            className="flex-1 font-mono text-xs"
            placeholder="KEY"
            value={entry.key}
            onChange={(e) => {
              const next = [...entries]
              next[i] = { ...next[i], key: e.target.value }
              onChange(next)
            }}
            disabled={disabled}
          />
          <Input
            className="flex-1 text-xs"
            placeholder="value"
            value={entry.value}
            onChange={(e) => {
              const next = [...entries]
              next[i] = { ...next[i], value: e.target.value }
              onChange(next)
            }}
            disabled={disabled}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-8 shrink-0"
            disabled={disabled}
            onClick={() =>
              entries.length > 1
                ? onChange(entries.filter((_, j) => j !== i))
                : onChange([{ key: "", value: "" }])
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
        disabled={disabled}
        onClick={() => onChange([...entries, { key: "", value: "" }])}
      >
        <PlusIcon className="mr-1 size-3.5" />
        Add variable
      </Button>
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export function ResourceCreationPage({
  envId,
  projectId,
}: {
  envId: string
  projectId: string
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const createMutation = useCreateResource(envId, projectId)
  const createFromGitMutation = useCreateResourceFromGit(envId, projectId)
  const checkRepoMutation = useCheckRepo()
  const { data: sourceConnections } = useGetSourceList()

  // ── Phase & flow ──────────────────────────────────────────────────────────
  const [phase, setPhase] = useState<Phase>("picker")
  const [flowType, setFlowType] = useState<FlowType>("preset")
  const [search, setSearch] = useState("")
  const [selectedPreset, setSelectedPreset] = useState<ResourcePreset | null>(
    null
  )

  // ── Docker / preset form ──────────────────────────────────────────────────
  const [name, setName] = useState("")
  const [nameError, setNameError] = useState("")
  const [tag, setTag] = useState("latest")
  const [customImage, setCustomImage] = useState("")
  const [resourceType, setResourceType] = useState("db")
  const [ports, setPorts] = useState<PortEntry[]>([{ host: "", container: "" }])
  const [envEntries, setEnvEntries] = useState<EnvEntry[]>([
    { key: "", value: "" },
  ])

  // ── Git form ──────────────────────────────────────────────────────────────
  const [gitUrl, setGitUrl] = useState("")
  const [gitBranch, setGitBranch] = useState("")
  const [gitToken, setGitToken] = useState("")
  const [buildMode, setBuildMode] = useState<"auto" | "dockerfile">("auto")
  const [imageTag, setImageTag] = useState("")
  const [gitName, setGitName] = useState("")
  const [selectedSourceConnectionId, setSelectedSourceConnectionId] =
    useState("")
  const [selectedRepoFullName, setSelectedRepoFullName] = useState("")
  const [gitPorts, setGitPorts] = useState<PortEntry[]>([
    { host: "", container: "" },
  ])
  const [gitEnv, setGitEnv] = useState<EnvEntry[]>([{ key: "", value: "" }])
  const [repoChecked, setRepoChecked] = useState<{
    available: boolean
    defaultBranch?: string
  } | null>(null)
  const { data: sourceRepos, isLoading: sourceReposLoading } =
    useGetSourceRepos(selectedSourceConnectionId)
  const selectedRepo =
    sourceRepos?.find((repo) => repo.full_name === selectedRepoFullName) ?? null
  const { data: sourceBranches, isLoading: sourceBranchesLoading } =
    useGetSourceBranches(
      selectedSourceConnectionId,
      selectedRepo?.owner ?? "",
      selectedRepo?.name ?? ""
    )

  // ── Filtering ─────────────────────────────────────────────────────────────
  const q = search.toLowerCase()
  const filtered = RESOURCE_PRESETS.filter(
    (p) =>
      !q ||
      p.name.toLowerCase().includes(q) ||
      p.id.toLowerCase().includes(q) ||
      p.image.toLowerCase().includes(q)
  )
  const dbPresets = filtered.filter((p) => p.type === "db")
  const servicePresets = filtered.filter(
    (p) => p.type === "service" || p.type === "app"
  )
  const showApps =
    !q ||
    "git".includes(q) ||
    "docker".includes(q) ||
    "repository".includes(q) ||
    "image".includes(q) ||
    "private".includes(q)

  // ── Handlers ──────────────────────────────────────────────────────────────
  const goBack = () => {
    if (phase === "config" || phase === "git") {
      setPhase("picker")
    } else if (projectId) {
      navigate({ to: "/projects/$projectId", params: { projectId } })
    } else {
      navigate({ to: "/projects" })
    }
  }

  const selectPreset = (preset: ResourcePreset) => {
    setSelectedPreset(preset)
    setFlowType("preset")
    setTag(preset.tags[0] ?? "latest")
    setName(preset.id)
    setResourceType(preset.type)
    setPorts(
      preset.ports.map((p) => ({ host: p.host, container: p.container }))
    )
    setEnvEntries(
      preset.env.length > 0
        ? preset.env.map((e) => ({ key: e.key, value: e.value }))
        : [{ key: "", value: "" }]
    )
    setPhase("config")
  }

  const selectDockerImage = () => {
    setSelectedPreset(null)
    setFlowType("docker")
    setTag("latest")
    setName("")
    setResourceType("app")
    setPorts([{ host: "", container: "" }])
    setEnvEntries([{ key: "", value: "" }])
    setPhase("config")
  }

  const selectPrivateRepository = () => {
    setSelectedPreset(null)
    setFlowType("git-private")
    setGitUrl("")
    setGitBranch("")
    setGitToken("")
    setBuildMode("auto")
    setImageTag("")
    setGitName("")
    setSelectedRepoFullName("")
    setSelectedSourceConnectionId(sourceConnections?.[0]?.id ?? "")
    setGitPorts([{ host: "", container: "" }])
    setGitEnv([{ key: "", value: "" }])
    setRepoChecked(null)
    setPhase("git")
  }

  const handleCheckRepo = () => {
    if (!gitUrl.trim()) return
    checkRepoMutation.mutate(
      {
        url: gitUrl.trim(),
        branch: gitBranch.trim() || undefined,
        token: gitToken.trim() || undefined,
      },
      {
        onSuccess: (res) => {
          setRepoChecked({
            available: res.available,
            defaultBranch: res.default_branch,
          })
          if (res.available && res.default_branch && !gitBranch) {
            setGitBranch(res.default_branch)
          }
          if (!res.available) {
            toast.error(res.error || "Repository not accessible")
          } else {
            toast.success("Repository verified!")
          }
        },
      }
    )
  }

  const handleSubmit = () => {
    if (flowType === "git" || flowType === "git-private") {
      const isPrivateRepoFlow = flowType === "git-private"
      if (!gitName.trim()) {
        toast.error("Resource name is required")
        return
      }
      if (!gitUrl.trim()) {
        toast.error("Repository URL is required")
        return
      }
      if (isPrivateRepoFlow && !selectedSourceConnectionId) {
        toast.error("Source connection is required")
        return
      }
      if (!imageTag.trim()) {
        toast.error("Image tag is required")
        return
      }
      setPhase("submitting")
      const portList = gitPorts
        .filter((p) => p.container.trim())
        .map((p) => ({
          host_port: parseInt(p.host.trim() || "0", 10),
          internal_port: parseInt(p.container.trim(), 10),
          proto: "tcp",
          label: "",
        }))
      const envVars = gitEnv
        .filter((e) => e.key.trim())
        .map((e) => ({ key: e.key.trim(), value: e.value, is_secret: false }))
      createFromGitMutation.mutate(
        {
          name: gitName.trim(),
          connection_id: isPrivateRepoFlow
            ? selectedSourceConnectionId || undefined
            : undefined,
          git_url: gitUrl.trim(),
          git_branch: gitBranch.trim() || undefined,
          build_mode: buildMode,
          git_token: isPrivateRepoFlow
            ? undefined
            : gitToken.trim() || undefined,
          image_tag: imageTag.trim(),
          ports: portList,
          env_vars: envVars,
        },
        {
          onSuccess: (resource) => {
            queryClient.invalidateQueries({
              queryKey: PROJECT_QUERY_KEYS.project(projectId),
            })
            toast.success(
              "Resource saved. Trigger a build from the detail page."
            )
            navigate({
              to: "/resources/$resourceId",
              params: { resourceId: resource.id },
            })
          },
          onError: () => {
            setPhase("git")
          },
        }
      )
    } else {
      if (!name.trim()) {
        setNameError(t("validation.required") || "Required")
        return
      }
      setNameError("")
      const image =
        flowType === "docker"
          ? customImage.trim()
          : (selectedPreset?.image ?? "")
      if (!image) {
        toast.error("Image is required")
        return
      }
      setPhase("submitting")
      const portList = ports
        .filter((p) => p.container.trim())
        .map((p) => ({
          host_port: parseInt(p.host.trim() || "0", 10),
          internal_port: parseInt(p.container.trim(), 10),
          proto: "tcp",
          label: "",
        }))
      const envVars = envEntries
        .filter((e) => e.key.trim())
        .map((e) => ({ key: e.key.trim(), value: e.value, is_secret: false }))
      createMutation.mutate(
        {
          name: name.trim(),
          type: resourceType,
          image,
          tag,
          ports: portList,
          env_vars: envVars,
        },
        {
          onSuccess: () => {
            queryClient.invalidateQueries({
              queryKey: PROJECT_QUERY_KEYS.project(projectId),
            })
            toast.success(t("projects.resource.created"))
            navigate({
              to: "/projects/$projectId",
              params: { projectId },
            })
          },
          onError: () => {
            setPhase("config")
          },
        }
      )
    }
  }

  const busy = phase === "submitting"

  // ── Title ─────────────────────────────────────────────────────────────────
  const title =
    phase === "picker"
      ? "New Resource"
      : phase === "git"
        ? flowType === "git-private"
          ? "Private Repository"
          : "Git Repository"
        : selectedPreset
          ? selectedPreset.name
          : flowType === "docker"
            ? "Docker Image"
            : "New Resource"

  // ── Layout ────────────────────────────────────────────────────────────────
  return (
    <div className="mx-auto flex flex-col gap-6 p-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-8 shrink-0"
          onClick={goBack}
          disabled={busy}
        >
          <ArrowLeftIcon className="size-4" />
        </Button>
        <div>
          <p className="text-xs text-muted-foreground">New Resource</p>
          <h1 className="text-lg font-semibold">{title}</h1>
        </div>
      </div>

      {/* ── Picker ─────────────────────────────────────────────── */}
      {phase === "picker" && (
        <div className="flex flex-col gap-6">
          {/* Search */}
          <div className="relative">
            <SearchIcon className="absolute top-2.5 left-3 size-4 text-muted-foreground" />
            <Input
              placeholder="Search databases, services…"
              className="pl-9"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>

          {/* Applications */}
          {showApps && (
            <section className="flex flex-col gap-3">
              <p className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                Applications
              </p>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {(!q ||
                  "git".includes(q) ||
                  "repository".includes(q) ||
                  "public".includes(q)) && (
                  <button
                    type="button"
                    onClick={() => {
                      setFlowType("git")
                      setSelectedSourceConnectionId("")
                      setSelectedRepoFullName("")
                      setPhase("git")
                    }}
                    className="flex flex-col gap-2 rounded-xl border bg-card p-3 text-left transition-shadow hover:border-primary/40 hover:shadow-sm"
                  >
                    <div className="flex items-center gap-2">
                      <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-violet-600 text-white">
                        <GitBranchIcon className="size-4" />
                      </div>
                      <div>
                        <p className="text-sm font-semibold">Git Repository</p>
                        <p className="text-xs text-muted-foreground">
                          public or private
                        </p>
                      </div>
                    </div>
                    <p className="line-clamp-2 text-xs text-muted-foreground">
                      Auto-detect stack or use existing Dockerfile. Builds &amp;
                      deploys automatically.
                    </p>
                  </button>
                )}
                {(!q ||
                  "private".includes(q) ||
                  "repo".includes(q) ||
                  "github".includes(q)) && (
                  <button
                    type="button"
                    onClick={selectPrivateRepository}
                    className="flex flex-col gap-2 rounded-xl border bg-card p-3 text-left transition-shadow hover:border-primary/40 hover:shadow-sm"
                  >
                    <div className="flex items-center gap-2">
                      <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
                        <GithubIcon className="size-4" />
                      </div>
                      <div>
                        <p className="text-sm font-semibold">
                          Private Repository
                        </p>
                        <p className="text-xs text-muted-foreground">
                          connected source
                        </p>
                      </div>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      Choose a repository from a connected GitHub App source,
                      then pick its branch.
                    </p>
                  </button>
                )}
                {(!q ||
                  "docker".includes(q) ||
                  "image".includes(q) ||
                  "custom".includes(q)) && (
                  <button
                    type="button"
                    onClick={selectDockerImage}
                    className="flex flex-col gap-2 rounded-xl border border-dashed bg-card p-3 text-left transition-shadow hover:border-primary/40 hover:shadow-sm"
                  >
                    <div className="flex items-center gap-2">
                      <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-sky-600 text-white">
                        <ServerIcon className="size-4" />
                      </div>
                      <div>
                        <p className="text-sm font-semibold">Docker Image</p>
                        <p className="text-xs text-muted-foreground">
                          any registry
                        </p>
                      </div>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      Use any Docker image from Docker Hub or a custom registry.
                    </p>
                  </button>
                )}
              </div>
            </section>
          )}

          {/* Databases */}
          {dbPresets.length > 0 && (
            <section className="flex flex-col gap-3">
              <p className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                Databases
              </p>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {dbPresets.map((p) => (
                  <PresetCard
                    key={p.id}
                    preset={p}
                    onClick={() => selectPreset(p)}
                  />
                ))}
              </div>
            </section>
          )}

          {/* Services */}
          {servicePresets.length > 0 && (
            <section className="flex flex-col gap-3">
              <p className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                Services
              </p>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {servicePresets.map((p) => (
                  <PresetCard
                    key={p.id}
                    preset={p}
                    onClick={() => selectPreset(p)}
                  />
                ))}
              </div>
            </section>
          )}

          {dbPresets.length === 0 &&
            servicePresets.length === 0 &&
            !showApps && (
              <p className="py-12 text-center text-sm text-muted-foreground">
                No results for &quot;{search}&quot;
              </p>
            )}
        </div>
      )}

      {/* ── Git form ───────────────────────────────────────────── */}
      {(phase === "git" ||
        (phase === "submitting" &&
          (flowType === "git" || flowType === "git-private"))) && (
        <div className="flex flex-col gap-5 rounded-xl border bg-card p-6">
          {flowType === "git-private" ? (
            !sourceConnections || sourceConnections.length === 0 ? (
              <div className="rounded-xl border border-dashed bg-muted/40 p-4">
                <p className="font-medium">No sources connected</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  Connect a GitHub App source first, then come back here to
                  choose a private repository.
                </p>
                <Button
                  type="button"
                  className="mt-4"
                  variant="outline"
                  onClick={() => navigate({ to: "/sources" })}
                >
                  Open Sources
                </Button>
              </div>
            ) : (
              <>
                <div className="grid gap-5 sm:grid-cols-2">
                  <div className="flex flex-col gap-1.5">
                    <Label>Source</Label>
                    <Select
                      value={selectedSourceConnectionId}
                      onValueChange={(value) => {
                        setSelectedSourceConnectionId(value)
                        setSelectedRepoFullName("")
                        setGitUrl("")
                        setGitBranch("")
                        setRepoChecked(null)
                      }}
                      disabled={busy}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="Choose source" />
                      </SelectTrigger>
                      <SelectContent>
                        {sourceConnections.map((source) => (
                          <SelectItem key={source.id} value={source.id}>
                            {source.display_name} · {source.account_identifier}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <Label>Repository</Label>
                    <Select
                      value={selectedRepoFullName}
                      onValueChange={(value) => {
                        const repo =
                          sourceRepos?.find(
                            (item) => item.full_name === value
                          ) ?? null
                        setSelectedRepoFullName(value)
                        setGitUrl(repo?.clone_url ?? "")
                        setRepoChecked(
                          repo
                            ? {
                                available: true,
                                defaultBranch: repo.default_ref,
                              }
                            : null
                        )
                        setGitBranch(repo?.default_ref ?? "")
                        setGitName((current) =>
                          current.trim() !== "" ? current : (repo?.name ?? "")
                        )
                      }}
                      disabled={
                        busy ||
                        !selectedSourceConnectionId ||
                        sourceReposLoading ||
                        !sourceRepos ||
                        sourceRepos.length === 0
                      }
                    >
                      <SelectTrigger>
                        <SelectValue
                          placeholder={
                            sourceReposLoading
                              ? "Loading repositories…"
                              : "Choose repository"
                          }
                        />
                      </SelectTrigger>
                      <SelectContent>
                        {(sourceRepos ?? []).map((repo) => (
                          <SelectItem
                            key={repo.full_name}
                            value={repo.full_name}
                          >
                            {repo.full_name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="grid gap-5 sm:grid-cols-2">
                  <div className="flex flex-col gap-1.5">
                    <Label>Branch</Label>
                    <Select
                      value={gitBranch}
                      onValueChange={setGitBranch}
                      disabled={
                        busy ||
                        !selectedRepo ||
                        sourceBranchesLoading ||
                        !sourceBranches ||
                        sourceBranches.length === 0
                      }
                    >
                      <SelectTrigger>
                        <SelectValue
                          placeholder={
                            sourceBranchesLoading
                              ? "Loading branches…"
                              : "Choose branch"
                          }
                        />
                      </SelectTrigger>
                      <SelectContent>
                        {(sourceBranches ?? []).map((branch) => (
                          <SelectItem key={branch.name} value={branch.name}>
                            {branch.name}
                            {branch.is_default ? " (default)" : ""}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <Label>Clone URL</Label>
                    <Input
                      value={gitUrl}
                      disabled
                      className="text-muted-foreground"
                    />
                  </div>
                </div>
              </>
            )
          ) : (
            <>
              <div className="flex flex-col gap-1.5">
                <Label>Repository URL</Label>
                <div className="flex gap-2">
                  <Input
                    placeholder="https://github.com/user/repo"
                    value={gitUrl}
                    onChange={(e) => {
                      setGitUrl(e.target.value)
                      setRepoChecked(null)
                    }}
                    disabled={busy}
                    className="flex-1"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    disabled={
                      !gitUrl.trim() || checkRepoMutation.isPending || busy
                    }
                    onClick={handleCheckRepo}
                  >
                    {checkRepoMutation.isPending ? "Checking…" : "Check"}
                  </Button>
                </div>
                {repoChecked && (
                  <p
                    className={`text-xs ${repoChecked.available ? "text-green-600" : "text-destructive"}`}
                  >
                    {repoChecked.available
                      ? `✓ Accessible${repoChecked.defaultBranch ? ` · default branch: ${repoChecked.defaultBranch}` : ""}`
                      : "✗ Not accessible"}
                  </p>
                )}
              </div>
            </>
          )}

          <div className="grid gap-5 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label>
                Branch <span className="text-muted-foreground">(optional)</span>
              </Label>
              <Input
                placeholder={repoChecked?.defaultBranch ?? "main"}
                value={gitBranch}
                onChange={(e) => setGitBranch(e.target.value)}
                disabled={busy}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              {flowType === "git-private" ? (
                <>
                  <Label>Connection</Label>
                  <Input
                    value={
                      sourceConnections?.find(
                        (source) => source.id === selectedSourceConnectionId
                      )?.account_identifier ?? ""
                    }
                    disabled
                    className="text-muted-foreground"
                  />
                </>
              ) : (
                <>
                  <Label>
                    Access Token{" "}
                    <span className="text-muted-foreground">
                      (private repos)
                    </span>
                  </Label>
                  <Input
                    type="password"
                    placeholder="ghp_xxxxxxxxxxxx"
                    value={gitToken}
                    onChange={(e) => setGitToken(e.target.value)}
                    disabled={busy}
                  />
                </>
              )}
            </div>
          </div>

          <div className="grid gap-5 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label>Build Mode</Label>
              <Select
                value={buildMode}
                onValueChange={(v) => setBuildMode(v as "auto" | "dockerfile")}
                disabled={busy}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">
                    Auto-detect (recommended)
                  </SelectItem>
                  <SelectItem value="dockerfile">
                    Use existing Dockerfile
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label>
                Image Tag{" "}
                <span className="text-xs text-muted-foreground">
                  (where to push)
                </span>
              </Label>
              <Input
                placeholder="ttl.sh/myapp:1h"
                value={imageTag}
                onChange={(e) => setImageTag(e.target.value)}
                disabled={busy}
              />
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Resource Name</Label>
            <Input
              placeholder="my-app"
              value={gitName}
              onChange={(e) => setGitName(e.target.value)}
              disabled={busy}
            />
          </div>

          <PortList ports={gitPorts} onChange={setGitPorts} disabled={busy} />
          <EnvList entries={gitEnv} onChange={setGitEnv} disabled={busy} />

          {/* Footer */}
          <div className="flex justify-end gap-2 border-t pt-4">
            <Button variant="outline" onClick={goBack} disabled={busy}>
              Back
            </Button>
            <Button onClick={handleSubmit} disabled={busy}>
              {busy ? "Saving…" : "Save"}
            </Button>
          </div>
        </div>
      )}

      {/* ── Docker / Preset config form ────────────────────────── */}
      {(phase === "config" ||
        (phase === "submitting" && flowType !== "git")) && (
        <div className="flex flex-col gap-5 rounded-xl border bg-card p-6">
          {/* Selected preset badge */}
          {selectedPreset && (
            <div className="flex items-center gap-3 rounded-lg bg-muted/50 p-3">
              <div
                className="flex size-9 shrink-0 items-center justify-center rounded-lg text-xs font-bold text-white"
                style={{ backgroundColor: selectedPreset.color }}
              >
                {selectedPreset.abbr}
              </div>
              <div>
                <p className="text-sm font-semibold">{selectedPreset.name}</p>
                <p className="font-mono text-xs text-muted-foreground">
                  {selectedPreset.image}
                </p>
              </div>
            </div>
          )}

          {/* Docker image input */}
          {flowType === "docker" && (
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

          <div className="grid gap-5 sm:grid-cols-2">
            {/* Type */}
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

            {/* Tag */}
            {selectedPreset ? (
              <div className="flex flex-col gap-1.5">
                <Label>Version</Label>
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
          </div>

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

          <PortList ports={ports} onChange={setPorts} disabled={busy} />
          <EnvList
            entries={envEntries}
            onChange={setEnvEntries}
            disabled={busy}
          />

          {/* Footer */}
          <div className="flex justify-end gap-2 border-t pt-4">
            <Button variant="outline" onClick={goBack} disabled={busy}>
              Back
            </Button>
            <Button onClick={handleSubmit} disabled={busy}>
              {busy
                ? t("databases.deploy.creating_btn")
                : t("projects.addResource")}
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
