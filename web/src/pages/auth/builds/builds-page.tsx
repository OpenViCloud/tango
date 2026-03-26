import { useEffect, useRef, useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"

import type { BuildJobModel, CreateBuildJobModel } from "@/@types/models"
import {
  useGetBuildJobList,
  useCreateBuildJob,
  useCancelBuildJob,
} from "@/hooks/api/use-build"
import { useBuildLogs } from "@/hooks/api/use-build-logs"
import { createBuildJobSchema } from "@/@types/models/build"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import { appIcons, actionIcons } from "@/lib/icons"

const BuildsIcon = appIcons.builds
const CreateIcon = actionIcons.create
const RefreshIcon = actionIcons.refresh
const StopIcon = actionIcons.stop

// ── Status badge ─────────────────────────────────────────────────────────────

const STATUS_VARIANT: Record<string, "default" | "secondary" | "outline"> = {
  queued: "secondary",
  cloning: "outline",
  detecting: "outline",
  generating: "outline",
  building: "default",
  done: "default",
  failed: "outline",
  canceled: "secondary",
}

const STATUS_COLOR: Record<string, string> = {
  queued: "text-muted-foreground",
  cloning: "text-blue-500",
  detecting: "text-blue-500",
  generating: "text-blue-500",
  building: "text-yellow-500",
  done: "text-green-500",
  failed: "text-destructive",
  canceled: "text-muted-foreground",
}

function StatusBadge({ status }: { status: string }) {
  return (
    <Badge variant={STATUS_VARIANT[status] ?? "secondary"}>
      <span className={STATUS_COLOR[status]}>{status}</span>
    </Badge>
  )
}

const ACTIVE_STATUSES = ["queued", "cloning", "detecting", "generating", "building"]

// ── New build form ────────────────────────────────────────────────────────────

function NewBuildSheet({
  open,
  onClose,
}: {
  open: boolean
  onClose: () => void
}) {
  const { t } = useTranslation()
  const createMutation = useCreateBuildJob()

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CreateBuildJobModel>({
    resolver: zodResolver(createBuildJobSchema),
    defaultValues: {
      git_url: "https://github.com/golang/example",
      git_branch: "master",
      image_tag: "ttl.sh/tango-test:1h",
    },
  })

  const onSubmit = async (data: CreateBuildJobModel) => {
    await createMutation.mutateAsync(data)
    toast.success(t("builds.toasts.submitted"))
    reset()
    onClose()
  }

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent className="w-full sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{t("builds.form.title")}</SheetTitle>
        </SheetHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="mt-6 flex flex-col gap-5">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="git_url">{t("builds.form.gitUrl")}</Label>
            <Input
              id="git_url"
              placeholder="https://github.com/user/repo"
              {...register("git_url")}
            />
            {errors.git_url && (
              <p className="text-destructive text-sm">{errors.git_url.message}</p>
            )}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="git_branch">
              {t("builds.form.gitBranch")}{" "}
              <span className="text-muted-foreground text-xs">({t("builds.form.optional")})</span>
            </Label>
            <Input id="git_branch" placeholder="main" {...register("git_branch")} />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="image_tag">{t("builds.form.imageTag")}</Label>
            <Input
              id="image_tag"
              placeholder="ghcr.io/user/app:v1"
              {...register("image_tag")}
            />
            {errors.image_tag && (
              <p className="text-destructive text-sm">{errors.image_tag.message}</p>
            )}
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" onClick={onClose}>
              {t("builds.form.cancel")}
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? t("builds.form.submitting") : t("builds.form.submit")}
            </Button>
          </div>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ── Log viewer sheet ──────────────────────────────────────────────────────────

function LogSheet({
  job,
  onClose,
}: {
  job: BuildJobModel | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  const { logs, status, connected } = useBuildLogs(job?.id ?? null)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [logs])

  const displayStatus = status ?? job?.status
  const isEmpty = !logs && !connected

  return (
    <Sheet open={Boolean(job)} onOpenChange={(v) => !v && onClose()}>
      <SheetContent className="w-full sm:max-w-2xl flex flex-col">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            {t("builds.logs.title")}
            {displayStatus && <StatusBadge status={displayStatus} />}
            {connected && (
              <span className="text-xs text-muted-foreground animate-pulse">
                {t("builds.logs.streaming")}
              </span>
            )}
          </SheetTitle>
          {job && (
            <p className="text-muted-foreground text-xs font-mono truncate">{job.image_tag}</p>
          )}
        </SheetHeader>

        <div className="flex-1 overflow-auto mt-4">
          {isEmpty ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-5/6" />
            </div>
          ) : (
            <pre className="bg-muted rounded-md p-4 text-xs font-mono whitespace-pre-wrap break-all leading-relaxed min-h-[200px]">
              {logs || t("builds.logs.empty")}
              {connected && <span className="animate-pulse">▌</span>}
              <div ref={bottomRef} />
            </pre>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

// ── Job row ───────────────────────────────────────────────────────────────────

function JobRow({
  job,
  onViewLogs,
  onCancel,
  cancelPending,
}: {
  job: BuildJobModel
  onViewLogs: (job: BuildJobModel) => void
  onCancel: (id: string) => void
  cancelPending: boolean
}) {
  const { t } = useTranslation()
  const isActive = ACTIVE_STATUSES.includes(job.status)

  return (
    <tr className="border-b last:border-0 hover:bg-muted/40 transition-colors">
      <td className="py-3 px-4">
        <span className="font-mono text-xs text-muted-foreground">{job.id}</span>
      </td>
      <td className="py-3 px-4">
        <StatusBadge status={job.status} />
      </td>
      <td className="py-3 px-4 max-w-[220px]">
        <span className="text-sm truncate block" title={job.git_url}>
          {job.git_url}
        </span>
        <span className="text-xs text-muted-foreground">{job.git_branch}</span>
      </td>
      <td className="py-3 px-4 max-w-[200px]">
        <span className="font-mono text-xs truncate block" title={job.image_tag}>
          {job.image_tag}
        </span>
      </td>
      <td className="py-3 px-4">
        <span className="text-xs text-muted-foreground">
          {new Date(job.created_at).toLocaleString()}
        </span>
      </td>
      <td className="py-3 px-4">
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => onViewLogs(job)}>
            {t("builds.actions.viewLogs")}
          </Button>
          {isActive && (
            <Button
              size="sm"
              variant="ghost"
              disabled={cancelPending}
              onClick={() => onCancel(job.id)}
            >
              <StopIcon className="size-3.5" />
              {t("builds.actions.cancel")}
            </Button>
          )}
        </div>
      </td>
    </tr>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export default function BuildsPage() {
  const { t } = useTranslation()
  const [pageIndex] = useState(0)
  const [showNewBuild, setShowNewBuild] = useState(false)
  const [selectedJob, setSelectedJob] = useState<BuildJobModel | null>(null)

  const { data, isLoading, isFetching, refetch } = useGetBuildJobList({
    pageIndex,
    pageSize: 20,
  })

  const cancelMutation = useCancelBuildJob()

  const jobs = data?.items ?? []
  const total = data?.totalItems ?? 0

  const handleCancel = async (id: string) => {
    await cancelMutation.mutateAsync(id)
    toast.success(t("builds.toasts.canceled"))
  }

  return (
    <>
      <PageHeaderCard
        icon={<BuildsIcon className="size-5" />}
        title={t("builds.page.title")}
        description={t("builds.page.description")}
        titleMeta={total}
        headerRight={
          <Button onClick={() => setShowNewBuild(true)}>
            <CreateIcon data-icon="inline-start" />
            {t("builds.actions.newBuild")}
          </Button>
        }
      />

      <SectionCard>
        <div className="flex flex-col gap-4">
          {/* Toolbar */}
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={isFetching}
              onClick={() => refetch()}
            >
              <RefreshIcon className="size-3.5" />
              {t("builds.actions.refresh")}
            </Button>
          </div>

          {/* Table */}
          <div className="rounded-md border overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="py-3 px-4 text-left font-medium">{t("builds.table.id")}</th>
                  <th className="py-3 px-4 text-left font-medium">{t("builds.table.status")}</th>
                  <th className="py-3 px-4 text-left font-medium">{t("builds.table.gitUrl")}</th>
                  <th className="py-3 px-4 text-left font-medium">{t("builds.table.imageTag")}</th>
                  <th className="py-3 px-4 text-left font-medium">{t("builds.table.createdAt")}</th>
                  <th className="py-3 px-4 text-left font-medium">{t("builds.table.actions")}</th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  Array.from({ length: 4 }).map((_, i) => (
                    <tr key={i} className="border-b last:border-0">
                      {Array.from({ length: 6 }).map((_, j) => (
                        <td key={j} className="py-3 px-4">
                          <Skeleton className="h-4 w-full" />
                        </td>
                      ))}
                    </tr>
                  ))
                ) : jobs.length === 0 ? (
                  <tr>
                    <td
                      colSpan={6}
                      className="py-10 text-center text-muted-foreground text-sm"
                    >
                      {t("builds.empty")}
                    </td>
                  </tr>
                ) : (
                  jobs.map((job) => (
                    <JobRow
                      key={job.id}
                      job={job}
                      onViewLogs={setSelectedJob}
                      onCancel={handleCancel}
                      cancelPending={cancelMutation.isPending}
                    />
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </SectionCard>

      <NewBuildSheet open={showNewBuild} onClose={() => setShowNewBuild(false)} />
      <LogSheet job={selectedJob} onClose={() => setSelectedJob(null)} />
    </>
  )
}
