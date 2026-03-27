import { useEffect, useRef, useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"

import type {
  BuildJobModel,
  CreateBuildJobModel,
  UploadBuildJobModel,
} from "@/@types/models"
import {
  useGetBuildJobList,
  useCreateBuildJob,
  useUploadBuildJob,
  useCancelBuildJob,
} from "@/hooks/api/use-build"
import { useBuildLogs } from "@/hooks/api/use-build-logs"
import {
  createBuildJobSchema,
  uploadBuildJobSchema,
} from "@/@types/models/build"
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
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
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

const ACTIVE_STATUSES = [
  "queued",
  "cloning",
  "detecting",
  "generating",
  "building",
]

// ── Build mode radio ──────────────────────────────────────────────────────────

function BuildModeField({
  value,
  onChange,
}: {
  value: "auto" | "dockerfile"
  onChange: (v: "auto" | "dockerfile") => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{t("builds.form.buildMode")}</Label>
      <div className="flex gap-3">
        {(["auto", "dockerfile"] as const).map((mode) => (
          <button
            key={mode}
            type="button"
            onClick={() => onChange(mode)}
            className={`flex-1 rounded-lg border px-3 py-2 text-left text-sm transition-colors ${
              value === mode
                ? "border-primary bg-primary/5 text-primary"
                : "border-border hover:border-primary/40"
            }`}
          >
            <p className="font-medium">
              {mode === "auto"
                ? t("builds.form.modeAuto")
                : t("builds.form.modeDockerfile")}
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {mode === "auto"
                ? t("builds.form.modeAutoDesc")
                : t("builds.form.modeDockerfileDesc")}
            </p>
          </button>
        ))}
      </div>
    </div>
  )
}

// ── New build sheet ────────────────────────────────────────────────────────────

function NewBuildSheet({
  open,
  onClose,
}: {
  open: boolean
  onClose: () => void
}) {
  const { t } = useTranslation()
  const createMutation = useCreateBuildJob()
  const uploadMutation = useUploadBuildJob()

  // Git form
  const gitForm = useForm<CreateBuildJobModel>({
    resolver: zodResolver(createBuildJobSchema),
    defaultValues: {
      git_url: "",
      git_branch: "",
      build_mode: "auto",
      image_tag: "",
    },
  })

  // Upload form
  const uploadForm = useForm<UploadBuildJobModel>({
    resolver: zodResolver(uploadBuildJobSchema),
    defaultValues: { image_tag: "", build_mode: "auto" },
  })
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [fileError, setFileError] = useState("")

  const handleClose = () => {
    gitForm.reset()
    uploadForm.reset()
    setSelectedFile(null)
    setFileError("")
    onClose()
  }

  const onGitSubmit = async (data: CreateBuildJobModel) => {
    await createMutation.mutateAsync(data)
    toast.success(t("builds.toasts.submitted"))
    handleClose()
  }

  const onUploadSubmit = async (data: UploadBuildJobModel) => {
    if (!selectedFile) {
      setFileError(t("builds.form.fileRequired"))
      return
    }
    await uploadMutation.mutateAsync({
      file: selectedFile,
      imageTag: data.image_tag,
      buildMode: data.build_mode,
    })
    toast.success(t("builds.toasts.submitted"))
    handleClose()
  }

  return (
    <Sheet open={open} onOpenChange={(v) => !v && handleClose()}>
      <SheetContent className="flex w-full flex-col sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{t("builds.form.title")}</SheetTitle>
        </SheetHeader>

        <Tabs defaultValue="git" className="mt-4 flex flex-1 flex-col px-4">
          <TabsList className="w-full">
            <TabsTrigger value="git" className="flex-1">
              {t("builds.form.tabGit")}
            </TabsTrigger>
            <TabsTrigger value="upload" className="flex-1">
              {t("builds.form.tabUpload")}
            </TabsTrigger>
          </TabsList>

          {/* ── Git tab ── */}
          <TabsContent value="git">
            <form
              onSubmit={gitForm.handleSubmit(onGitSubmit)}
              className="flex flex-col gap-5 pt-2"
            >
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="git_url">{t("builds.form.gitUrl")}</Label>
                <Input
                  id="git_url"
                  placeholder="https://github.com/user/repo"
                  {...gitForm.register("git_url")}
                />
                {gitForm.formState.errors.git_url && (
                  <p className="text-sm text-destructive">
                    {gitForm.formState.errors.git_url.message}
                  </p>
                )}
              </div>

              <div className="flex flex-col gap-1.5">
                <Label htmlFor="git_branch">
                  {t("builds.form.gitBranch")}{" "}
                  <span className="text-xs text-muted-foreground">
                    ({t("builds.form.optional")})
                  </span>
                </Label>
                <Input
                  id="git_branch"
                  placeholder="main"
                  {...gitForm.register("git_branch")}
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <Label htmlFor="git_image_tag">
                  {t("builds.form.imageTag")}
                </Label>
                <Input
                  id="git_image_tag"
                  placeholder="ghcr.io/user/app:v1"
                  {...gitForm.register("image_tag")}
                />
                {gitForm.formState.errors.image_tag && (
                  <p className="text-sm text-destructive">
                    {gitForm.formState.errors.image_tag.message}
                  </p>
                )}
              </div>

              <BuildModeField
                value={gitForm.watch("build_mode") ?? "auto"}
                onChange={(v) => gitForm.setValue("build_mode", v)}
              />

              <div className="flex justify-end gap-2 pt-2">
                <Button type="button" variant="outline" onClick={handleClose}>
                  {t("builds.form.cancel")}
                </Button>
                <Button type="submit" disabled={createMutation.isPending}>
                  {createMutation.isPending
                    ? t("builds.form.submitting")
                    : t("builds.form.submit")}
                </Button>
              </div>
            </form>
          </TabsContent>

          {/* ── Upload tab ── */}
          <TabsContent value="upload">
            <form
              onSubmit={uploadForm.handleSubmit(onUploadSubmit)}
              className="flex flex-col gap-5 pt-2"
            >
              <div className="flex flex-col gap-1.5">
                <Label>{t("builds.form.archiveFile")}</Label>
                <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-border p-6 transition-colors hover:border-primary/40">
                  <input
                    type="file"
                    accept=".zip,.tar.gz,.tgz"
                    className="hidden"
                    onChange={(e) => {
                      const f = e.target.files?.[0] ?? null
                      setSelectedFile(f)
                      setFileError("")
                    }}
                  />
                  {selectedFile ? (
                    <div className="text-center">
                      <p className="max-w-[280px] truncate text-sm font-medium">
                        {selectedFile.name}
                      </p>
                      <p className="mt-0.5 text-xs text-muted-foreground">
                        {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                      </p>
                    </div>
                  ) : (
                    <div className="text-center">
                      <p className="text-sm text-muted-foreground">
                        {t("builds.form.dropArchive")}
                      </p>
                      <p className="mt-1 text-xs text-muted-foreground">
                        .zip, .tar.gz, .tgz
                      </p>
                    </div>
                  )}
                </label>
                {fileError && (
                  <p className="text-sm text-destructive">{fileError}</p>
                )}
              </div>

              <div className="flex flex-col gap-1.5">
                <Label htmlFor="upload_image_tag">
                  {t("builds.form.imageTag")}
                </Label>
                <Input
                  id="upload_image_tag"
                  placeholder="ghcr.io/user/app:v1"
                  {...uploadForm.register("image_tag")}
                />
                {uploadForm.formState.errors.image_tag && (
                  <p className="text-sm text-destructive">
                    {uploadForm.formState.errors.image_tag.message}
                  </p>
                )}
              </div>

              <BuildModeField
                value={uploadForm.watch("build_mode") ?? "auto"}
                onChange={(v) => uploadForm.setValue("build_mode", v)}
              />

              <div className="flex justify-end gap-2 pt-2">
                <Button type="button" variant="outline" onClick={handleClose}>
                  {t("builds.form.cancel")}
                </Button>
                <Button type="submit" disabled={uploadMutation.isPending}>
                  {uploadMutation.isPending
                    ? t("builds.form.submitting")
                    : t("builds.form.submit")}
                </Button>
              </div>
            </form>
          </TabsContent>
        </Tabs>
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
      <SheetContent className="flex w-full flex-col sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            {t("builds.logs.title")}
            {displayStatus && <StatusBadge status={displayStatus} />}
            {connected && (
              <span className="animate-pulse text-xs text-muted-foreground">
                {t("builds.logs.streaming")}
              </span>
            )}
          </SheetTitle>
          {job && (
            <p className="truncate font-mono text-xs text-muted-foreground">
              {job.image_tag}
            </p>
          )}
        </SheetHeader>

        <div className="mt-4 flex-1 overflow-auto">
          {isEmpty ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-5/6" />
            </div>
          ) : (
            <pre className="min-h-[200px] rounded-md bg-muted p-4 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap">
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
  const source =
    job.source_type === "upload" ? (job.archive_name ?? "upload") : job.git_url

  return (
    <tr className="border-b transition-colors last:border-0 hover:bg-muted/40">
      <td className="px-4 py-3">
        <span className="font-mono text-xs text-muted-foreground">
          {job.id}
        </span>
      </td>
      <td className="px-4 py-3">
        <StatusBadge status={job.status} />
      </td>
      <td className="max-w-[220px] px-4 py-3">
        <span className="block truncate text-sm" title={source}>
          {source}
        </span>
        <span className="text-xs text-muted-foreground">
          {job.source_type === "upload" ? job.build_mode : job.git_branch}
        </span>
      </td>
      <td className="max-w-[200px] px-4 py-3">
        <span
          className="block truncate font-mono text-xs"
          title={job.image_tag}
        >
          {job.image_tag}
        </span>
      </td>
      <td className="px-4 py-3">
        <span className="text-xs text-muted-foreground">
          {new Date(job.created_at).toLocaleString()}
        </span>
      </td>
      <td className="px-4 py-3">
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

          <div className="overflow-x-auto rounded-md border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-4 py-3 text-left font-medium">
                    {t("builds.table.id")}
                  </th>
                  <th className="px-4 py-3 text-left font-medium">
                    {t("builds.table.status")}
                  </th>
                  <th className="px-4 py-3 text-left font-medium">
                    {t("builds.table.source")}
                  </th>
                  <th className="px-4 py-3 text-left font-medium">
                    {t("builds.table.imageTag")}
                  </th>
                  <th className="px-4 py-3 text-left font-medium">
                    {t("builds.table.createdAt")}
                  </th>
                  <th className="px-4 py-3 text-left font-medium">
                    {t("builds.table.actions")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  Array.from({ length: 4 }).map((_, i) => (
                    <tr key={i} className="border-b last:border-0">
                      {Array.from({ length: 6 }).map((_, j) => (
                        <td key={j} className="px-4 py-3">
                          <Skeleton className="h-4 w-full" />
                        </td>
                      ))}
                    </tr>
                  ))
                ) : jobs.length === 0 ? (
                  <tr>
                    <td
                      colSpan={6}
                      className="py-10 text-center text-sm text-muted-foreground"
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

      <NewBuildSheet
        open={showNewBuild}
        onClose={() => setShowNewBuild(false)}
      />
      <LogSheet job={selectedJob} onClose={() => setSelectedJob(null)} />
    </>
  )
}
