import { zodResolver } from "@hookform/resolvers/zod"
import { Link, useNavigate } from "@tanstack/react-router"
import {
  ArrowLeftIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  FolderIcon,
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
  PROJECT_QUERY_KEYS,
  useCreateEnvironment,
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
