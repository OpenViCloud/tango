import {
  ArrowLeftIcon,
  ChevronDownIcon,
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
import { Checkbox } from "@/components/ui/checkbox"
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
  useCreateResourceStack,
  useCreateResourceFromGit,
  useGetResourceTemplates,
  useGetResourceStackTemplates,
} from "@/hooks/api/use-project"
import {
  useGetSourceBranches,
  useGetSourceList,
  useGetSourceRepos,
} from "@/hooks/api/use-source"
import { useSwarmNodes, useSwarmStatus } from "@/hooks/api/use-swarm"
import type { ResourceStackTemplateModel, ResourceTemplateModel } from "@/@types/models"
import { useNavigate } from "@tanstack/react-router"
import { useQueryClient } from "@tanstack/react-query"

// ── Types ─────────────────────────────────────────────────────────────────────

type EnvEntry = { key: string; value: string }
type PortEntry = { host: string; container: string }
type FlowType = "preset" | "docker" | "git" | "git-private" | "stack"
type Phase = "picker" | "config" | "git" | "stack" | "submitting"
type ResourcePreset = ResourceTemplateModel
type ResourceStack = ResourceStackTemplateModel

// ── Sub-components ────────────────────────────────────────────────────────────

function PresetVisual({
  preset,
  className = "h-10 w-10 rounded-lg",
}: {
  preset: ResourcePreset
  className?: string
}) {
  if (preset.icon_url) {
    return (
      <img
        src={preset.icon_url}
        className={`${className} shrink-0 object-contain`}
        alt={`${preset.name} logo`}
      />
    )
  }

  return (
    <div
      className={`flex shrink-0 items-center justify-center text-xs font-bold text-white ${className}`}
      style={{ backgroundColor: preset.color }}
    >
      {preset.abbr}
    </div>
  )
}

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
        <PresetVisual preset={preset} />

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

function StackVisual({
  stack,
  className = "h-10 w-10 rounded-lg",
}: {
  stack: ResourceStack
  className?: string
}) {
  if (stack.icon_url) {
    return (
      <img
        src={stack.icon_url}
        className={`${className} shrink-0 object-contain`}
        alt={`${stack.name} logo`}
      />
    )
  }

  return (
    <div
      className={`flex shrink-0 items-center justify-center text-xs font-bold text-white ${className}`}
      style={{ backgroundColor: stack.color }}
    >
      {stack.abbr}
    </div>
  )
}

function StackCard({
  stack,
  onClick,
}: {
  stack: ResourceStack
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex flex-col gap-2 rounded-xl border bg-card p-3 text-left transition-shadow hover:border-primary/40 hover:shadow-sm"
    >
      <div className="flex items-center gap-2">
        <StackVisual stack={stack} />

        <div>
          <p className="text-sm font-semibold">{stack.name}</p>
          <p className="text-xs text-muted-foreground">
            {stack.components.length} services
          </p>
        </div>
      </div>
      <p className="line-clamp-2 text-xs text-muted-foreground">
        {stack.description}
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
  const createStackMutation = useCreateResourceStack(envId, projectId)
  const createFromGitMutation = useCreateResourceFromGit(envId, projectId)
  const checkRepoMutation = useCheckRepo()
  const {
    data: resourceTemplates = [],
    isLoading: resourceTemplatesLoading,
    isError: resourceTemplatesError,
  } = useGetResourceTemplates()
  const {
    data: resourceStackTemplates = [],
    isLoading: resourceStackTemplatesLoading,
    isError: resourceStackTemplatesError,
  } = useGetResourceStackTemplates()
  const { data: sourceConnections } = useGetSourceList()

  // ── Phase & flow ──────────────────────────────────────────────────────────
  const [phase, setPhase] = useState<Phase>("picker")
  const [flowType, setFlowType] = useState<FlowType>("preset")
  const [search, setSearch] = useState("")
  const [selectedPreset, setSelectedPreset] = useState<ResourcePreset | null>(
    null
  )
  const [selectedStack, setSelectedStack] = useState<ResourceStack | null>(null)

  // ── Swarm ─────────────────────────────────────────────────────────────────
  const { data: swarmStatus } = useSwarmStatus()
  const isSwarmManager = swarmStatus?.is_manager ?? false
  const { data: swarmNodes = [] } = useSwarmNodes()
  const [nodeId, setNodeId] = useState<string>("")
  const [replicas, setReplicas] = useState(1)

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
  const [stackNamePrefix, setStackNamePrefix] = useState("")
  const [stackImage, setStackImage] = useState("")
  const [stackTag, setStackTag] = useState("latest")
  const [stackEnvEntries, setStackEnvEntries] = useState<EnvEntry[]>([
    { key: "", value: "" },
  ])
  const [stackComponents, setStackComponents] = useState<
    {
      id: string
      type: string
      cmd: string
      port: string
      volumes: string[]
      env: EnvEntry[]
      expanded: boolean
    }[]
  >([])

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
  const filtered = resourceTemplates.filter(
    (p) =>
      !q ||
      p.name.toLowerCase().includes(q) ||
      p.id.toLowerCase().includes(q) ||
      p.image.toLowerCase().includes(q)
  )
  const filteredStacks = resourceStackTemplates.filter(
    (stack) =>
      !q ||
      stack.name.toLowerCase().includes(q) ||
      stack.id.toLowerCase().includes(q) ||
      stack.image.toLowerCase().includes(q)
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
    setSelectedStack(null)
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
    setSelectedStack(null)
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
    setSelectedStack(null)
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

  const selectStack = (stack: ResourceStack) => {
    setSelectedPreset(null)
    setSelectedStack(stack)
    setFlowType("stack")
    setStackNamePrefix(stack.id)
    setStackImage(stack.image)
    setStackTag(stack.tags[0] ?? "latest")
    setStackEnvEntries(
      stack.shared_env.length > 0
        ? stack.shared_env.map((entry) => ({ key: entry.key, value: entry.value }))
        : [{ key: "", value: "" }]
    )
    setStackComponents(
      stack.components.map((c) => ({
        id: c.id,
        type: c.type,
        cmd: (c.cmd ?? []).join(" "),
        port:
          c.ports.length > 0
            ? `${c.ports[0].host}:${c.ports[0].container}`
            : "",
        volumes: c.volumes ?? [],
        env: c.env.length > 0 ? c.env.map((e) => ({ key: e.key, value: e.value })) : [],
        expanded: false,
      }))
    )
    setPhase("stack")
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
    if (flowType === "stack") {
      if (!selectedStack) {
        toast.error("Stack template is required")
        return
      }
      if (!stackNamePrefix.trim()) {
        toast.error("Stack name prefix is required")
        return
      }
      setPhase("submitting")
      createStackMutation.mutate(
        {
          template_id: selectedStack.id,
          name_prefix: stackNamePrefix.trim(),
          image: stackImage.trim() || undefined,
          tag: stackTag.trim() || undefined,
          node_id: nodeId.trim() || null,
          shared_env_vars: stackEnvEntries
            .filter((entry) => entry.key.trim())
            .map((entry) => ({
              key: entry.key.trim(),
              value: entry.value,
              is_secret: false,
            })),
          custom_components: stackComponents
            .filter((c) => c.id.trim())
            .map((c) => {
              const cmd = c.cmd.trim().split(/\s+/).filter(Boolean)
              const portMatch = c.port.trim().match(/^(\d+):(\d+)$/)
              return {
                id: c.id.trim(),
                type: c.type || "service",
                cmd,
                ports: portMatch
                  ? [
                      {
                        host_port: parseInt(portMatch[1], 10),
                        internal_port: parseInt(portMatch[2], 10),
                        proto: "tcp",
                      },
                    ]
                  : [],
                volumes: c.volumes,
                env: c.env
                  .filter((e) => e.key.trim())
                  .map((e) => ({ key: e.key.trim(), value: e.value, is_secret: false })),
              }
            }),
        },
        {
          onSuccess: (result) => {
            queryClient.invalidateQueries({
              queryKey: PROJECT_QUERY_KEYS.project(projectId),
            })
            toast.success(
              `Created ${result.resources.length} resources for ${selectedStack.name}`
            )
            navigate({
              to: "/projects/$projectId",
              params: { projectId },
            })
          },
          onError: () => {
            setPhase("stack")
          },
        }
      )
    } else if (flowType === "git" || flowType === "git-private") {
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
      const presetConfig: Record<string, unknown> = {}
      if (selectedPreset?.volumes?.length) {
        presetConfig.volumes = selectedPreset.volumes
      }
      if (selectedPreset?.cmd?.length) {
        presetConfig.cmd = selectedPreset.cmd
      }
      createMutation.mutate(
        {
          name: name.trim(),
          type: resourceType,
          image,
          tag,
          config: Object.keys(presetConfig).length ? presetConfig : undefined,
          node_id: nodeId.trim() || null,
          replicas: isSwarmManager ? Math.max(1, replicas) : 1,
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
        : phase === "stack"
          ? (selectedStack?.name ?? "Resource Stack")
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

          {resourceTemplatesLoading && (
            <p className="text-sm text-muted-foreground">
              Loading resource templates...
            </p>
          )}

          {resourceStackTemplatesLoading && (
            <p className="text-sm text-muted-foreground">
              Loading resource stacks...
            </p>
          )}

          {resourceTemplatesError && (
            <p className="rounded-xl border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
              Could not load resource templates.
            </p>
          )}

          {resourceStackTemplatesError && (
            <p className="rounded-xl border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
              Could not load resource stacks.
            </p>
          )}

          {filteredStacks.length > 0 && (
            <section className="flex flex-col gap-3">
              <p className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                Stacks
              </p>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {filteredStacks.map((stack) => (
                  <StackCard
                    key={stack.id}
                    stack={stack}
                    onClick={() => selectStack(stack)}
                  />
                ))}
              </div>
            </section>
          )}

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

          {filteredStacks.length === 0 &&
            dbPresets.length === 0 &&
            servicePresets.length === 0 &&
            !showApps && (
              <p className="py-12 text-center text-sm text-muted-foreground">
                No results for &quot;{search}&quot;
              </p>
            )}
        </div>
      )}

      {(phase === "stack" ||
        (phase === "submitting" && flowType === "stack")) && selectedStack && (
        <div className="flex flex-col gap-5 rounded-xl border bg-card p-6">
          <div className="flex items-center gap-3 rounded-lg bg-muted/50 p-3">
            <StackVisual stack={selectedStack} className="h-10 w-10 rounded-lg" />
            <div>
              <p className="text-sm font-semibold">{selectedStack.name}</p>
              <p className="font-mono text-xs text-muted-foreground">
                {stackImage}:{stackTag}
              </p>
            </div>
          </div>

          <div className="grid gap-5 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label>Stack Name Prefix</Label>
              <Input
                placeholder="airflow"
                value={stackNamePrefix}
                onChange={(e) => setStackNamePrefix(e.target.value)}
                disabled={busy}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label>Version</Label>
              <Input
                list={`stack-tags-${selectedStack.id}`}
                placeholder="e.g. 3.0.2"
                value={stackTag}
                onChange={(e) => setStackTag(e.target.value)}
                disabled={busy}
              />
              <datalist id={`stack-tags-${selectedStack.id}`}>
                {selectedStack.tags.map((value) => (
                  <option key={value} value={value} />
                ))}
              </datalist>
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Image</Label>
            <Input
              value={stackImage}
              onChange={(e) => setStackImage(e.target.value)}
              disabled={busy}
            />
          </div>

          {isSwarmManager && swarmNodes.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <Label>Node</Label>
              <Select
                value={nodeId || "__any__"}
                onValueChange={(value) =>
                  setNodeId(value === "__any__" ? "" : value)
                }
                disabled={busy}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Any node" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__any__">Any node</SelectItem>
                  {swarmNodes.map((node) => (
                    <SelectItem key={node.id} value={node.id}>
                      {node.hostname}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <Label>Components</Label>
              <button
                type="button"
                onClick={() =>
                  setStackComponents((prev) => [
                    ...prev,
                    { id: "", type: "service", cmd: "", port: "", volumes: [], env: [], expanded: true },
                  ])
                }
                disabled={busy}
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                <PlusIcon className="size-3" />
                Add
              </button>
            </div>

            <div className="flex flex-col gap-2">
              {stackComponents.map((comp, idx) => {
                const update = (patch: Partial<typeof comp>) =>
                  setStackComponents((prev) =>
                    prev.map((c, i) => (i === idx ? { ...c, ...patch } : c))
                  )
                return (
                  <div key={idx} className="rounded-lg border overflow-hidden">
                    {/* Header row */}
                    <div className="grid grid-cols-[1fr_1fr_auto_auto_auto] gap-2 items-center p-2">
                      <div className="flex items-center gap-1.5 min-w-0">
                        {comp.type === "job" && (
                          <span className="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-semibold bg-amber-500/15 text-amber-600">
                            Init
                          </span>
                        )}
                        <Input
                          placeholder="ID (e.g. db-migrate)"
                          value={comp.id}
                          onChange={(e) => update({ id: e.target.value })}
                          disabled={busy}
                        />
                      </div>
                      <Input
                        placeholder="Command (e.g. db migrate)"
                        value={comp.cmd}
                        onChange={(e) => update({ cmd: e.target.value })}
                        disabled={busy}
                      />
                      {comp.type === "job" ? (
                        <div className="w-32" />
                      ) : (
                        <Input
                          placeholder="Port (e.g. 8080:8080)"
                          value={comp.port}
                          onChange={(e) => update({ port: e.target.value })}
                          disabled={busy}
                          className="w-32"
                        />
                      )}
                      <button
                        type="button"
                        onClick={() => update({ expanded: !comp.expanded })}
                        disabled={busy}
                        className="text-muted-foreground hover:text-foreground transition-colors"
                        title="Edit env & volumes"
                      >
                        <ChevronDownIcon
                          className={`size-4 transition-transform ${comp.expanded ? "rotate-180" : ""}`}
                        />
                      </button>
                      <button
                        type="button"
                        onClick={() =>
                          setStackComponents((prev) => prev.filter((_, i) => i !== idx))
                        }
                        disabled={busy}
                        className="text-muted-foreground hover:text-destructive transition-colors"
                      >
                        <MinusIcon className="size-4" />
                      </button>
                    </div>

                    {/* Expanded section — env & volumes */}
                    {comp.expanded && (
                      <div className="border-t bg-muted/20 p-3 space-y-4">
                        {/* Volumes */}
                        <div className="flex flex-col gap-1.5">
                          <span className="text-xs font-medium text-muted-foreground">
                            Volumes
                          </span>
                          {comp.volumes.map((vol, vi) => (
                            <div key={vi} className="flex gap-1.5 items-center">
                              <Input
                                className="flex-1 font-mono text-xs"
                                placeholder="host/path:/container/path"
                                value={vol}
                                onChange={(e) => {
                                  const next = [...comp.volumes]
                                  next[vi] = e.target.value
                                  update({ volumes: next })
                                }}
                                disabled={busy}
                              />
                              <button
                                type="button"
                                onClick={() =>
                                  update({ volumes: comp.volumes.filter((_, j) => j !== vi) })
                                }
                                disabled={busy}
                                className="text-muted-foreground hover:text-destructive transition-colors"
                              >
                                <MinusIcon className="size-3.5" />
                              </button>
                            </div>
                          ))}
                          <button
                            type="button"
                            onClick={() => update({ volumes: [...comp.volumes, ""] })}
                            disabled={busy}
                            className="flex items-center gap-1 self-start text-xs text-muted-foreground hover:text-foreground transition-colors"
                          >
                            <PlusIcon className="size-3" /> Add volume
                          </button>
                        </div>

                        {/* Env vars */}
                        <div className="flex flex-col gap-1.5">
                          <span className="text-xs font-medium text-muted-foreground">
                            Env Vars
                          </span>
                          {comp.env.map((e, ei) => (
                            <div key={ei} className="flex gap-1.5 items-center">
                              <Input
                                className="flex-1 font-mono text-xs"
                                placeholder="KEY"
                                value={e.key}
                                onChange={(ev) => {
                                  const next = [...comp.env]
                                  next[ei] = { ...next[ei], key: ev.target.value }
                                  update({ env: next })
                                }}
                                disabled={busy}
                              />
                              <Input
                                className="flex-1 text-xs"
                                placeholder="value"
                                value={e.value}
                                onChange={(ev) => {
                                  const next = [...comp.env]
                                  next[ei] = { ...next[ei], value: ev.target.value }
                                  update({ env: next })
                                }}
                                disabled={busy}
                              />
                              <button
                                type="button"
                                onClick={() =>
                                  update({ env: comp.env.filter((_, j) => j !== ei) })
                                }
                                disabled={busy}
                                className="text-muted-foreground hover:text-destructive transition-colors"
                              >
                                <MinusIcon className="size-3.5" />
                              </button>
                            </div>
                          ))}
                          <button
                            type="button"
                            onClick={() =>
                              update({ env: [...comp.env, { key: "", value: "" }] })
                            }
                            disabled={busy}
                            className="flex items-center gap-1 self-start text-xs text-muted-foreground hover:text-foreground transition-colors"
                          >
                            <PlusIcon className="size-3" /> Add env var
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                )
              })}

              {stackComponents.length === 0 && (
                <p className="text-xs text-muted-foreground">
                  No components. Click Add to define one.
                </p>
              )}
            </div>
          </div>

          <EnvList
            entries={stackEnvEntries}
            onChange={setStackEnvEntries}
            disabled={busy}
          />

          <div className="rounded-lg border border-dashed bg-muted/30 p-4 text-sm text-muted-foreground">
            The API will create one resource per enabled component. Required
            components are always created.
          </div>

          <div className="flex justify-end gap-2 border-t pt-4">
            <Button variant="outline" onClick={goBack} disabled={busy}>
              Back
            </Button>
            <Button onClick={handleSubmit} disabled={busy}>
              {busy ? "Creating…" : "Create Stack"}
            </Button>
          </div>
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
              <PresetVisual preset={selectedPreset} className="h-10 w-10 rounded-lg" />
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
                <Input
                  list={`preset-tags-${selectedPreset.id}`}
                  placeholder="e.g. latest"
                  value={tag}
                  onChange={(e) => setTag(e.target.value)}
                  disabled={busy}
                />
                <datalist id={`preset-tags-${selectedPreset.id}`}>
                  {selectedPreset.tags.map((t) => (
                    <option key={t} value={t} />
                  ))}
                </datalist>
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

          {/* Swarm controls — visible whenever this node is a swarm manager */}
          {isSwarmManager && (
            <div className="grid gap-5 sm:grid-cols-2">
              {/* Node selector — only when nodes are loaded */}
              {swarmNodes.length > 0 ? (
                <div className="flex flex-col gap-1.5">
                  <Label>Deploy on node</Label>
                  <Select
                    value={nodeId || "__any__"}
                    onValueChange={(v) => setNodeId(v === "__any__" ? "" : v)}
                    disabled={busy}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Any node (auto)" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__any__">Any node (auto)</SelectItem>
                      {swarmNodes.map((node) => (
                        <SelectItem key={node.id} value={node.id}>
                          {node.hostname}
                          <span className="ml-2 text-xs text-muted-foreground">
                            {node.role} · {node.state}
                          </span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              ) : (
                <div /> /* empty cell to keep replicas in second column */
              )}
              <div className="flex flex-col gap-1.5">
                <Label>
                  Replicas{" "}
                  <span className="text-xs text-muted-foreground">
                    (swarm tasks)
                  </span>
                </Label>
                <Input
                  type="number"
                  min={1}
                  value={replicas}
                  onChange={(e) =>
                    setReplicas(Math.max(1, parseInt(e.target.value, 10) || 1))
                  }
                  disabled={busy}
                />
              </div>
            </div>
          )}

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
