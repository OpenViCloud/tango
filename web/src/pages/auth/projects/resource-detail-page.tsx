import { useEffect, useMemo, useRef, useState } from "react"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"

import type { ResourceRunModel } from "@/@types/models"
import { Badge } from "@/components/ui/badge"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import {
  useGetResource,
  useGetResourceEnvVars,
  useSetResourceEnvVars,
  useStartResource,
  useStopResource,
  PROJECT_QUERY_KEYS,
} from "@/hooks/api/use-project"
import { useResourceRunLogs } from "@/hooks/api/use-resource-run-logs"
import ResourceDetails from "@/pages/auth/projects/components/resource-details"
import type { EnvEntry } from "@/pages/auth/projects/components/ConfigGeneralForm"
import { useQueryClient } from "@tanstack/react-query"

type ResourceDetailPageProps = {
  resourceId: string
}

export default function ResourceDetailPage({
  resourceId,
}: ResourceDetailPageProps) {
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
  const startMutation = useStartResource()
  const stopMutation = useStopResource()

  const [activeRun, setActiveRun] = useState<ResourceRunModel | null>(null)
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

  const handleSave = async (entries: EnvEntry[]) => {
    try {
      await setEnvVarsMutation.mutateAsync(
        entries
          .filter((item) => item.key.trim())
          .map((item) => ({
            key: item.key.trim(),
            value: item.value,
            is_secret: item.is_secret,
          }))
      )
      toast.success(t("projects.resource.updated"))
    } catch {
      // Mutation toast handles backend errors.
    }
  }

  const handleStart = () => {
    startMutation.mutate(resourceId, {
      onSuccess: (run) => setActiveRun(run),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleStop = () => {
    stopMutation.mutate(resourceId, {
      onSuccess: async () => {
        toast.success(t("projects.resource.stopped"))
        await queryClient.invalidateQueries({
          queryKey: ["resource", resourceId],
        })
        await queryClient.invalidateQueries({
          queryKey: PROJECT_QUERY_KEYS.resourceEnvVars(resourceId),
        })
      },
      onError: (err) => toast.error(err.message),
    })
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
        pending={setEnvVarsMutation.isPending}
        actionPending={
          startMutation.isPending ||
          stopMutation.isPending ||
          activeRun !== null
        }
        isLoadingEnvVars={isLoadingEnvVars}
        isEnvVarsError={isEnvVarsError}
      />
      <ResourceRunLogSheet
        run={activeRun}
        resourceName={resource.name}
        onClose={() => setActiveRun(null)}
        onCompleted={async () => {
          await queryClient.invalidateQueries({
            queryKey: ["resource", resourceId],
          })
          await queryClient.invalidateQueries({
            queryKey: PROJECT_QUERY_KEYS.resourceEnvVars(resourceId),
          })
        }}
      />
    </>
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
      toast.success(t("projects.resource.started"))
      void onCompleted()
      return
    }
    if (status === "failed") {
      toast.error(run?.error_msg || t("projects.resource.runFailed"))
      void onCompleted()
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
