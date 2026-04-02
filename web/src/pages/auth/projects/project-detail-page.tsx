import { zodResolver } from "@hookform/resolvers/zod"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  AlertTriangleIcon,
  ArrowLeftIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  GitForkIcon,
} from "lucide-react"
import React, { useEffect, useRef, useState } from "react"
import { useForm } from "react-hook-form"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import type {
  CreateEnvironmentModel,
  EnvironmentModel,
  ResourceModel,
  ResourceRunModel,
  ResourceTemplateModel,
} from "@/@types/models"
import { createEnvironmentSchema } from "@/@types/models/project"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import {
  PROJECT_QUERY_KEYS,
  useCreateEnvironment,
  useDeleteEnvironment,
  useDeleteResource,
  useForkEnvironment,
  useGetProject,
  useGetResourceTemplates,
  useStartResource,
  useStopResource,
} from "@/hooks/api/use-project"
import { useResourceRunLogs } from "@/hooks/api/use-resource-run-logs"
import { actionIcons, appIcons } from "@/lib/icons"
import { resolveResourceVisual } from "@/lib/resource-visual"
import { Route } from "@/routes/_auth/projects/$projectId"
import { useQueryClient } from "@tanstack/react-query"

const ProjectsIcon = appIcons.projects
const StartIcon = actionIcons.start
const StopIcon = actionIcons.stop
const DeleteIcon = actionIcons.delete
const CreateIcon = actionIcons.create
const EditIcon = actionIcons.edit

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
  conflictingHostPorts,
  templates,
}: {
  resource: ResourceModel
  onStart: (resource: ResourceModel) => void
  onStop: (id: string) => void
  onDelete: () => void
  busy: boolean
  conflictingHostPorts: Set<number>
  templates: ResourceTemplateModel[]
}) {
  const { t } = useTranslation()
  const isRunning = resource.status === "running"
  const hasContainer = !!resource.container_id
  const canStart = !isRunning
  const canStop = isRunning && hasContainer

  const portSummary = resource.ports
    .map((p) => `${p.host_port > 0 ? p.host_port : "?"}→${p.internal_port}`)
    .join(", ")

  const hasPortConflict = resource.ports.some(
    (p) => p.host_port > 0 && conflictingHostPorts.has(p.host_port)
  )
  const hasUnsetPort = resource.ports.some((p) => p.host_port === 0)
  const visual = resolveResourceVisual(resource, templates)

  return (
    <div
      className={`flex flex-col gap-3 rounded-xl border bg-card p-4 ${hasPortConflict ? "border-yellow-500/50" : ""}`}
    >
      <div className="flex items-start gap-3">
        {visual.iconUrl ? (
          <img
            src={visual.iconUrl}
            alt={`${resource.name} logo`}
            className="size-10 shrink-0 rounded-lg object-contain"
          />
        ) : (
          <div
            className="flex size-9 shrink-0 items-center justify-center rounded-lg text-xs font-bold text-white"
            style={{ backgroundColor: visual.color }}
          >
            {visual.abbr}
          </div>
        )}
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
        <div className="flex items-center gap-1.5">
          <p className="font-mono text-xs text-muted-foreground">
            {portSummary}
          </p>
          {(hasPortConflict || hasUnsetPort) && (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <AlertTriangleIcon className="size-3.5 shrink-0 text-yellow-500" />
                </TooltipTrigger>
                <TooltipContent>
                  {hasPortConflict
                    ? t("projects.resource.portConflict")
                    : t("projects.resource.portUnset")}
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )}
        </div>
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
          onClick={onDelete}
          className="text-destructive hover:text-destructive"
        >
          <DeleteIcon className="size-3.5" />
        </Button>
      </div>
    </div>
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
    })
  })

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="sm:max-w-sm">
        <SheetHeader>
          <SheetTitle>{t("projects.addEnv")}</SheetTitle>
        </SheetHeader>
        <form onSubmit={onSubmit} className="mt-4 flex flex-1 flex-col gap-4">
          <div className="flex flex-col gap-1.5 px-4">
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
          <SheetFooter className="mt-auto gap-2 border-t">
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

        <div className="mt-4 flex-1 overflow-auto px-4">
          {isEmpty ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-5/6" />
            </div>
          ) : (
            <pre className="min-h-55 rounded-md bg-muted p-4 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap">
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

// ── Fork environment dialog ───────────────────────────────────────────────────

function ForkEnvironmentDialog({
  env,
  projectId,
  open,
  onOpenChange,
}: {
  env: EnvironmentModel
  projectId: string
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { t } = useTranslation()
  const forkMutation = useForkEnvironment(projectId)
  const [name, setName] = useState(`${env.name}-fork`)

  const handleClose = (v: boolean) => {
    if (!v) setName(`${env.name}-fork`)
    onOpenChange(v)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    forkMutation.mutate(
      { envId: env.id, name: name.trim() },
      {
        onSuccess: () => {
          toast.success(t("projects.forkEnvSuccess"))
          handleClose(false)
        },
      }
    )
  }

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="sm:max-w-sm">
        <SheetHeader>
          <SheetTitle>{t("projects.forkEnvTitle")}</SheetTitle>
        </SheetHeader>
        <form
          onSubmit={handleSubmit}
          className="mt-4 flex flex-1 flex-col gap-4"
        >
          <div className="flex flex-col gap-1.5 px-4">
            <Label htmlFor="fork-env-name">
              {t("projects.forkEnvNameLabel")}
            </Label>
            <Input
              id="fork-env-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("projects.forkEnvNamePlaceholder")}
            />
          </div>
          <SheetFooter className="gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={forkMutation.isPending}
              onClick={() => handleClose(false)}
            >
              {t("projects.cancel")}
            </Button>
            <Button
              type="submit"
              disabled={forkMutation.isPending || !name.trim()}
            >
              {forkMutation.isPending
                ? t("projects.forking")
                : t("projects.fork")}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ── Delete environment confirm dialog ─────────────────────────────────────────

function DeleteEnvironmentDialog({
  env,
  open,
  onOpenChange,
  onConfirm,
  isDeleting,
}: {
  env: EnvironmentModel
  open: boolean
  onOpenChange: (v: boolean) => void
  onConfirm: () => void
  isDeleting: boolean
}) {
  const { t } = useTranslation()
  const [input, setInput] = useState("")

  const handleClose = (v: boolean) => {
    if (!v) setInput("")
    onOpenChange(v)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (input !== env.name) return
    onConfirm()
  }

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="sm:max-w-sm">
        <SheetHeader>
          <SheetTitle>{t("projects.deleteEnvTitle")}</SheetTitle>
        </SheetHeader>
        <form
          onSubmit={handleSubmit}
          className="mt-4 flex flex-1 flex-col gap-4"
        >
          <p className="mx-4 text-sm text-muted-foreground">
            {t("projects.deleteEnvConfirmHint")}{" "}
            <span className="font-medium text-foreground">{env.name}</span>{" "}
            {t("projects.deleteEnvConfirmHint2")}
          </p>
          <div className="flex flex-col gap-1.5 px-4">
            <Label htmlFor="delete-env-input">
              {t("projects.deleteEnvInputLabel")}
            </Label>
            <Input
              id="delete-env-input"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={env.name}
              autoComplete="off"
            />
          </div>
          <SheetFooter className="gap-2 border-t">
            <Button
              type="button"
              variant="outline"
              disabled={isDeleting}
              onClick={() => handleClose(false)}
            >
              {t("projects.cancel")}
            </Button>
            <Button
              type="submit"
              variant="destructive"
              disabled={isDeleting || input !== env.name}
            >
              {isDeleting
                ? t("projects.deleting")
                : t("projects.deleteEnvButton")}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

function DeleteResourceDialog({
  resource,
  open,
  onOpenChange,
  onConfirm,
  isDeleting,
}: {
  resource: ResourceModel | null
  open: boolean
  onOpenChange: (v: boolean) => void
  onConfirm: () => void
  isDeleting: boolean
}) {
  const [input, setInput] = useState("")

  const handleClose = (v: boolean) => {
    if (!v) setInput("")
    onOpenChange(v)
  }

  const expectedName = resource?.name ?? ""
  const canConfirm = Boolean(expectedName) && input === expectedName

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="sm:max-w-sm">
        <SheetHeader>
          <SheetTitle>Delete resource</SheetTitle>
        </SheetHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (!canConfirm) return
            onConfirm()
          }}
          className="mt-4 flex flex-1 flex-col gap-4"
        >
          <p className="mx-4 text-sm text-muted-foreground">
            Type{" "}
            <span className="font-medium text-foreground">{expectedName}</span>{" "}
            to confirm resource deletion.
          </p>
          <div className="flex flex-col gap-1.5 px-4">
            <Label htmlFor="delete-resource-input">Resource name</Label>
            <Input
              id="delete-resource-input"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={expectedName}
              autoComplete="off"
            />
          </div>
          <SheetFooter className="gap-2 border-t">
            <Button
              type="button"
              variant="outline"
              disabled={isDeleting}
              onClick={() => handleClose(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="destructive"
              disabled={isDeleting || !canConfirm}
            >
              {isDeleting ? "Deleting..." : "Delete resource"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ── Environment section ───────────────────────────────────────────────────────

function EnvironmentSection({
  env,
  projectId,
  conflictingHostPorts,
  templates,
}: {
  env: EnvironmentModel
  projectId: string
  conflictingHostPorts: Set<number>
  templates: ResourceTemplateModel[]
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(true)
  const [forkOpen, setForkOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [resourceToDelete, setResourceToDelete] = useState<ResourceModel | null>(null)
  const [activeRun, setActiveRun] = useState<ResourceRunModel | null>(null)
  const [activeRunResourceName, setActiveRunResourceName] = useState<
    string | null
  >(null)
  const startMutation = useStartResource()
  const stopMutation = useStopResource()
  const deleteMutation = useDeleteResource(projectId)
  const deleteEnvMutation = useDeleteEnvironment(projectId)

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
    })
  }

  const handleDelete = () => {
    if (!resourceToDelete) return
    deleteMutation.mutate(resourceToDelete.id, {
      onSuccess: () => toast.success(t("projects.resource.deleted")),
    })
  }

  const handleConfirmDeleteEnv = () => {
    deleteEnvMutation.mutate(env.id, {
      onSuccess: () => {
        toast.success(t("projects.deleteEnvSuccess"))
        setDeleteOpen(false)
      },
    })
  }

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
          <Button asChild size="sm" variant="outline">
            <Link
              to="/environments/$envId/resources/new"
              params={{ envId: env.id }}
              search={{ projectId }}
            >
              <CreateIcon className="mr-1 size-3.5" />
              {t("projects.addResource")}
            </Link>
          </Button>
          <Button size="sm" variant="outline" onClick={() => setForkOpen(true)}>
            <GitForkIcon className="mr-1 size-3.5" />
            {t("projects.fork")}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            disabled={deleteEnvMutation.isPending}
            onClick={() => setDeleteOpen(true)}
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
                  onDelete={() => setResourceToDelete(resource)}
                  busy={actionBusy}
                  conflictingHostPorts={conflictingHostPorts}
                  templates={templates}
                />
              ))}
            </div>
          )}
        </div>
      )}

      <ResourceRunLogSheet
        run={activeRun}
        resourceName={activeRunResourceName}
        onClose={() => {
          setActiveRun(null)
          setActiveRunResourceName(null)
        }}
        onCompleted={handleRunCompleted}
      />

      <ForkEnvironmentDialog
        env={env}
        projectId={projectId}
        open={forkOpen}
        onOpenChange={setForkOpen}
      />

      <DeleteEnvironmentDialog
        env={env}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onConfirm={handleConfirmDeleteEnv}
        isDeleting={deleteEnvMutation.isPending}
      />

      <DeleteResourceDialog
        resource={resourceToDelete}
        open={Boolean(resourceToDelete)}
        onOpenChange={(open) => {
          if (!open) setResourceToDelete(null)
        }}
        onConfirm={handleDelete}
        isDeleting={deleteMutation.isPending}
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
  const { data: resourceTemplates = [] } = useGetResourceTemplates()

  // Compute which host_ports appear in more than one resource across the project.
  const conflictingHostPorts = React.useMemo(() => {
    if (!project) return new Set<number>()
    const portCount = new Map<number, number>()
    for (const env of project.environments) {
      for (const resource of env.resources) {
        for (const p of resource.ports) {
          if (p.host_port > 0) {
            portCount.set(p.host_port, (portCount.get(p.host_port) ?? 0) + 1)
          }
        }
      }
    }
    const result = new Set<number>()
    for (const [port, count] of portCount) {
      if (count > 1) result.add(port)
    }
    return result
  }, [project])

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
            icon={<ProjectsIcon />}
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
                conflictingHostPorts={conflictingHostPorts}
                templates={resourceTemplates}
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
