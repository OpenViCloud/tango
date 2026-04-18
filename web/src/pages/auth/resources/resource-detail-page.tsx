import { useMemo, useRef, useState, useEffect } from "react"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"
import { Link } from "@tanstack/react-router"
import { ExternalLink } from "lucide-react"

import type { ResourceRunModel } from "@/@types/models"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import {
  PROJECT_QUERY_KEYS,
  useBuildResource,
  useGetResource,
  useGetResourceEnvVars,
  useRestartResource,
  useSetResourceEnvVars,
  useUpdateResource,
  useStartResource,
  useStopResource,
  useScaleResource,
} from "@/hooks/api/use-project"
import { useSwarmStatus } from "@/hooks/api/use-swarm"
import { useGetBuildJob } from "@/hooks/api/use-build"
import { useBuildLogs } from "@/hooks/api/use-build-logs"
import { useResourceRunLogs } from "@/hooks/api/use-resource-run-logs"
import ResourceDetails from "@/pages/auth/resources/components/resource-details"
import type { EnvEntry, PortEntry } from "@/pages/auth/resources/components/ConfigGeneralForm"
import type { VolumeEntry } from "@/pages/auth/resources/components/PersistentStorageForm"
import { useQueryClient } from "@tanstack/react-query"

type ResourceDetailPageProps = {
  resourceId: string
}

const BUILD_STATUS_COLOR: Record<string, string> = {
  queued:     "text-muted-foreground",
  cloning:    "text-blue-500",
  detecting:  "text-blue-500",
  generating: "text-blue-500",
  building:   "text-yellow-500",
  done:       "text-green-500",
  failed:     "text-destructive",
  canceled:   "text-muted-foreground",
}

export default function ResourceDetailPage({ resourceId }: ResourceDetailPageProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const {
    data: resource,
    isLoading: isLoadingResource,
    isError: isResourceError,
  } = useGetResource(resourceId)
  const {
    data: envVars,
    isLoading: isLoadingEnvVars,
    isError: isEnvVarsError,
  } = useGetResourceEnvVars(resourceId)
  const setEnvVarsMutation = useSetResourceEnvVars(resourceId)
  const updateResourceMutation = useUpdateResource(resourceId)
  const startMutation = useStartResource()
  const stopMutation = useStopResource()
  const restartMutation = useRestartResource()
  const scaleMutation = useScaleResource()
  const { data: swarmStatus } = useSwarmStatus()
  const isSwarmManager = swarmStatus?.is_manager ?? false
  const buildMutation = useBuildResource(resourceId)
  const [activeRun, setActiveRun] = useState<ResourceRunModel | null>(null)
  const [activeBuildJobId, setActiveBuildJobId] = useState<string | null>(null)
  const [runSuccessMessageKey, setRunSuccessMessageKey] = useState("projects.resource.started")

  const initialEnvEntries = useMemo<EnvEntry[]>(
    () =>
      envVars && envVars.length > 0
        ? envVars.map((item) => ({
            key: item.key,
            value: item.value ?? "",
            is_secret: item.is_secret,
          }))
        : [{ key: "", value: "", is_secret: false }],
    [envVars]
  )

  const handleSave = async (
    entries: EnvEntry[],
    name: string,
    ports: PortEntry[],
    volumes: VolumeEntry[],
    memoryLimit: number,
    cpuLimit: number
  ) => {
    try {
      await Promise.all([
        updateResourceMutation.mutateAsync({
          name: name.trim() || resource!.name,
          replicas: resource?.replicas ?? 1,
          memory_limit: memoryLimit,
          cpu_limit: cpuLimit,
          config: {
            ...(resource?.config ?? {}),
            volumes: volumes
              .filter((item) => item.source.trim() && item.target.trim())
              .map((item) =>
                item.mode === "ro"
                  ? `${item.source.trim()}:${item.target.trim()}:ro`
                  : `${item.source.trim()}:${item.target.trim()}`
              ),
          },
          ports: ports
            .filter((p) => p.internal_port !== "")
            .map((p) => ({
              host_port: p.host_port !== "" ? Number(p.host_port) : 0,
              internal_port: Number(p.internal_port),
              proto: p.proto || "tcp",
              label: p.label,
            })),
        }),
        setEnvVarsMutation.mutateAsync(
          entries
            .filter((item) => item.key.trim())
            .map((item) => ({
              key: item.key.trim(),
              value: item.value,
              is_secret: item.is_secret,
            }))
        ),
      ])
      toast.success(t("projects.resource.updated"))
    } catch {
      // Mutation toast handles backend errors.
    }
  }

  const handleStart = () => {
    setRunSuccessMessageKey("projects.resource.started")
    startMutation.mutate(resourceId, {
      onSuccess: (run) => setActiveRun(run),
    })
  }

  const handleBuild = () => {
    buildMutation.mutate(undefined, {
      onSuccess: (result) => {
        setActiveBuildJobId(result.build_job_id)
      },
    })
  }

  const handleStop = () => {
    stopMutation.mutate(resourceId, {
      onSuccess: async () => {
        toast.success(t("projects.resource.stopped"))
        await queryClient.invalidateQueries({ queryKey: ["resource", resourceId] })
        await queryClient.invalidateQueries({
          queryKey: PROJECT_QUERY_KEYS.resourceEnvVars(resourceId),
        })
      },
    })
  }

  const handleRestart = () => {
    setRunSuccessMessageKey("projects.resource.restarted")
    restartMutation.mutate(resourceId, {
      onSuccess: (run) => setActiveRun(run),
    })
  }

  const handleScale = (replicas: number) => {
    scaleMutation.mutate(
      { resourceId, replicas },
      {
        onSuccess: () => {
          toast.success(`Scaled to ${replicas} replica${replicas !== 1 ? "s" : ""}`)
          queryClient.invalidateQueries({ queryKey: PROJECT_QUERY_KEYS.resource(resourceId) })
        },
        onError: () => {
          toast.error("Failed to scale resource")
        },
      }
    )
  }

  if (isLoadingResource) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-12 w-72 rounded-xl" />
        <Skeleton className="h-52 w-full rounded-xl" />
      </div>
    )
  }

  if (isResourceError || !resource) {
    return (
      <div className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
        {t("projects.resource.loadFailed")}
      </div>
    )
  }

  return (
    <>
      <ResourceDetails
        key={`${resourceId}:${JSON.stringify(initialEnvEntries)}`}
        resource={resource}
        initialEnvEntries={initialEnvEntries}
        onSave={handleSave}
        onStart={handleStart}
        onStop={handleStop}
        onRestart={handleRestart}
        onBuild={handleBuild}
        onScale={handleScale}
        isSwarmManager={isSwarmManager}
        scalePending={scaleMutation.isPending}
        pending={setEnvVarsMutation.isPending || updateResourceMutation.isPending}
        actionPending={
          startMutation.isPending ||
          stopMutation.isPending ||
          restartMutation.isPending ||
          buildMutation.isPending ||
          activeRun !== null
        }
        isLoadingEnvVars={isLoadingEnvVars}
        isEnvVarsError={isEnvVarsError}
      />

      <BuildLogSheet
        buildJobId={activeBuildJobId}
        onClose={() => setActiveBuildJobId(null)}
        onCompleted={async () => {
          await queryClient.invalidateQueries({ queryKey: ["resource", resourceId] })
        }}
      />

      <ResourceRunLogSheet
        run={activeRun}
        resourceName={resource.name}
        successMessageKey={runSuccessMessageKey}
        onClose={() => setActiveRun(null)}
        onCompleted={async () => {
          await queryClient.invalidateQueries({ queryKey: ["resource", resourceId] })
          await queryClient.invalidateQueries({
            queryKey: PROJECT_QUERY_KEYS.resourceEnvVars(resourceId),
          })
        }}
      />
    </>
  )
}

// ── Build log sheet ────────────────────────────────────────────────────────────

function BuildLogSheet({
  buildJobId,
  onClose,
  onCompleted,
}: {
  buildJobId: string | null
  onClose: () => void
  onCompleted: () => void | Promise<void>
}) {
  const { logs, status: wsStatus, connected } = useBuildLogs(buildJobId)
  const { data: job } = useGetBuildJob(buildJobId)
  const bottomRef = useRef<HTMLDivElement>(null)
  const notifiedRef = useRef<string | null>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [logs])

  useEffect(() => {
    notifiedRef.current = null
  }, [buildJobId])

  useEffect(() => {
    const finalStatus = wsStatus ?? job?.status
    if (!finalStatus) return
    if (notifiedRef.current === finalStatus) return
    if (finalStatus !== "done" && finalStatus !== "failed" && finalStatus !== "canceled") return
    notifiedRef.current = finalStatus
    if (finalStatus === "done") {
      toast.success("Build completed successfully!")
    } else {
      toast.error(`Build ${finalStatus}`)
    }
    void onCompleted()
  }, [wsStatus, job?.status, onCompleted])

  const displayStatus = wsStatus ?? job?.status
  const isEmpty = !logs && !connected

  return (
    <Sheet open={Boolean(buildJobId)} onOpenChange={(v) => !v && onClose()}>
      <SheetContent className="flex w-full flex-col sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2 flex-wrap">
            <span>Build Logs</span>
            {displayStatus && (
              <Badge variant="outline" className="font-normal">
                <span className={BUILD_STATUS_COLOR[displayStatus] ?? ""}>{displayStatus}</span>
              </Badge>
            )}
            {connected && (
              <span className="animate-pulse text-xs text-muted-foreground">streaming…</span>
            )}
          </SheetTitle>

          {/* image tag + link to builds page */}
          <div className="flex items-center justify-between gap-2">
            {job?.image_tag && (
              <p className="truncate font-mono text-xs text-muted-foreground">{job.image_tag}</p>
            )}
            {buildJobId && (
              <Button asChild variant="ghost" size="sm" className="h-6 gap-1 px-2 text-xs shrink-0">
                <Link to="/builds">
                  <ExternalLink className="h-3 w-3" />
                  View in Builds
                </Link>
              </Button>
            )}
          </div>
        </SheetHeader>

        <div className="mt-4 flex-1 overflow-auto">
          {isEmpty ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-5/6" />
            </div>
          ) : (
            <pre className="min-h-[220px] whitespace-pre-wrap break-all rounded-md bg-muted p-4 font-mono text-xs leading-relaxed">
              {logs || "Waiting for build output…"}
              {connected && <span className="animate-pulse">▌</span>}
              <div ref={bottomRef} />
            </pre>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

// ── Run log sheet ──────────────────────────────────────────────────────────────

function ResourceRunLogSheet({
  run,
  resourceName,
  successMessageKey,
  onClose,
  onCompleted,
}: {
  run: ResourceRunModel | null
  resourceName: string | null
  successMessageKey: string
  onClose: () => void
  onCompleted: () => void | Promise<void>
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
      toast.success(t(successMessageKey))
      void onCompleted()
      return
    }
    if (status === "failed") {
      toast.error(run?.error_msg || t("projects.resource.runFailed"))
      void onCompleted()
    }
  }, [onCompleted, run?.error_msg, status, successMessageKey, t])

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
            <pre className="min-h-[220px] whitespace-pre-wrap break-all rounded-md bg-muted p-4 font-mono text-xs leading-relaxed">
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
