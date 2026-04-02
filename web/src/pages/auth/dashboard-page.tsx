import { Link } from "@tanstack/react-router"
import {
  ActivityIcon,
  AlertTriangleIcon,
  ArrowRightIcon,
  BoxIcon,
  FolderGit2Icon,
  GlobeIcon,
  PlayCircleIcon,
  ServerCogIcon,
  Settings2Icon,
} from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import type { BuildJobModel } from "@/@types/models"
import type { ResourceModel } from "@/@types/models/project"
import type { BaseDomainModel } from "@/services/api/base-domain-service"
import type { SettingsModel } from "@/services/api/settings-service"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { appIcons } from "@/lib/icons"
import { cn } from "@/lib/utils"
import { useGetBaseDomains } from "@/hooks/api/use-base-domains"
import { useGetBuildJobList } from "@/hooks/api/use-build"
import { useGetProjectList } from "@/hooks/api/use-project"
import { useReconcileResources } from "@/hooks/api/use-project"
import { useGetSettings } from "@/hooks/api/use-settings"
import { useGetSourceList } from "@/hooks/api/use-source"

const DashboardIcon = appIcons.dashboard

type AttentionTone = "danger" | "warning" | "neutral"

type DashboardResource = ResourceModel & {
  environmentName: string
  projectId: string
  projectName: string
}

type AttentionItem = {
  id: string
  title: string
  description: string
  href: string
  cta: string
  tone: AttentionTone
}

type ActivityItem = {
  id: string
  title: string
  description: string
  timestamp: string
  href: string
  status: string
  kind: "build" | "resource"
}

function formatDateTime(value?: string) {
  if (!value) return "Unknown"

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "Unknown"

  return new Intl.DateTimeFormat("vi-VN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date)
}

function formatCompactDate(value?: string) {
  if (!value) return "No activity"

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "No activity"

  return new Intl.DateTimeFormat("vi-VN", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date)
}

function formatBuildLabel(build: BuildJobModel) {
  if (build.source_type === "upload") {
    return build.archive_name || build.image_tag
  }

  return build.git_url || build.image_tag
}

function getBuildStatusTone(status: BuildJobModel["status"]) {
  if (status === "failed") return "warning"
  if (status === "done") return "success"
  if (status === "canceled") return "secondary"
  return "outline"
}

function getResourceStatusTone(status: string) {
  if (status === "running") return "success"
  if (status === "error") return "warning"
  if (status === "stopped") return "secondary"
  return "outline"
}

function getSetupStatus(settings?: SettingsModel, baseDomains: BaseDomainModel[] = [], sourcesCount = 0) {
  return [
    {
      label: "Git sources",
      value: sourcesCount > 0 ? `${sourcesCount} connected` : "Missing",
      done: sourcesCount > 0,
      href: "/sources",
    },
    {
      label: "Base domains",
      value: baseDomains.length > 0 ? `${baseDomains.length} configured` : "Missing",
      done: baseDomains.length > 0,
      href: "/domains",
    },
    {
      label: "App ingress",
      value: settings?.app_domain
        ? `${settings.app_tls_enabled ? "HTTPS" : "HTTP"} active`
        : "Missing",
      done: Boolean(settings?.app_domain),
      href: "/settings",
    },
  ]
}

function StatCard({
  label,
  value,
  hint,
  tone = "default",
}: {
  label: string
  value: string | number
  hint: string
  tone?: "default" | "danger"
}) {
  return (
    <div
      className={cn(
        "rounded-2xl border p-4",
        tone === "danger"
          ? "border-destructive/30 bg-destructive/5"
          : "border-border/70 bg-card/70"
      )}
    >
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="mt-2 text-3xl font-semibold tracking-tight">{value}</p>
      <p className="mt-1 text-sm text-muted-foreground">{hint}</p>
    </div>
  )
}

function AttentionCard({ item }: { item: AttentionItem }) {
  return (
    <div
      className={cn(
        "rounded-2xl border p-4",
        item.tone === "danger"
          ? "border-destructive/30 bg-destructive/5"
          : item.tone === "warning"
            ? "border-amber-500/30 bg-amber-500/5"
            : "border-border/70 bg-muted/20"
      )}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="font-medium">{item.title}</p>
          <p className="mt-1 text-sm text-muted-foreground">{item.description}</p>
        </div>
        <Button asChild size="sm" variant="outline">
          <Link to={item.href}>
            {item.cta}
            <ArrowRightIcon data-icon="inline-end" />
          </Link>
        </Button>
      </div>
    </div>
  )
}

function ActivityRow({ item }: { item: ActivityItem }) {
  return (
    <Link
      to={item.href}
      className="flex items-start justify-between gap-4 rounded-2xl border border-border/70 bg-background/70 p-4 transition-colors hover:bg-muted/40"
    >
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <Badge variant={item.kind === "build" ? "outline" : "secondary"}>
            {item.kind}
          </Badge>
          <Badge variant={item.kind === "build" ? getBuildStatusTone(item.status as BuildJobModel["status"]) : getResourceStatusTone(item.status)}>
            {item.status}
          </Badge>
        </div>
        <p className="mt-3 font-medium">{item.title}</p>
        <p className="mt-1 text-sm text-muted-foreground">{item.description}</p>
      </div>
      <p className="shrink-0 text-xs text-muted-foreground">
        {formatCompactDate(item.timestamp)}
      </p>
    </Link>
  )
}

function ResourceRow({ resource }: { resource: DashboardResource }) {
  return (
    <Link
      to="/resources/$resourceId"
      params={{ resourceId: resource.id }}
      className="flex items-start justify-between gap-4 rounded-2xl border border-border/70 bg-background/70 p-4 transition-colors hover:bg-muted/40"
    >
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <Badge variant="outline">{resource.type}</Badge>
          <Badge variant={getResourceStatusTone(resource.status)}>
            {resource.status}
          </Badge>
          {resource.source_type === "git" ? (
            <Badge variant="secondary">git</Badge>
          ) : null}
        </div>
        <p className="mt-3 font-medium">{resource.name}</p>
        <p className="mt-1 text-sm text-muted-foreground">
          {resource.projectName} / {resource.environmentName}
        </p>
      </div>
      <div className="shrink-0 text-right text-xs text-muted-foreground">
        <p>{resource.ports.length} ports</p>
        <p className="mt-1">{formatCompactDate(resource.updated_at)}</p>
      </div>
    </Link>
  )
}

export default function DashboardPage() {
  const { t } = useTranslation()
  const reconcileResources = useReconcileResources()
  const { data: projects, isLoading: projectsLoading } = useGetProjectList()
  const { data: buildData, isLoading: buildsLoading } = useGetBuildJobList({
    pageIndex: 1,
    pageSize: 8,
    orderBy: "created_at",
    ascending: false,
  })
  const { data: sources, isLoading: sourcesLoading } = useGetSourceList()
  const { data: settings, isLoading: settingsLoading } = useGetSettings()
  const { data: baseDomains = [], isLoading: domainsLoading } = useGetBaseDomains()

  const buildItems = buildData?.items ?? []
  const safeProjects = projects ?? []
  const safeSources = sources ?? []

  const environmentsCount = safeProjects.reduce(
    (total, project) => total + project.environments.length,
    0
  )

  const resources = safeProjects.flatMap((project) =>
    project.environments.flatMap((environment) =>
      environment.resources.map((resource) => ({
        ...resource,
        environmentName: environment.name,
        projectId: project.id,
        projectName: project.name,
      }))
    )
  )

  const runningResources = resources.filter((resource) => resource.status === "running")
  const unhealthyResources = resources.filter((resource) =>
    ["error", "stopped"].includes(resource.status)
  )
  const gitResources = resources.filter((resource) => resource.source_type === "git")

  const failedBuilds = buildItems.filter((build) => build.status === "failed")
  const activeBuilds = buildItems.filter((build) =>
    ["queued", "cloning", "detecting", "generating", "building"].includes(build.status)
  )

  const attentionItems: AttentionItem[] = [
    failedBuilds.length > 0
      ? {
          id: "failed-builds",
          title: `${failedBuilds.length} failed build${failedBuilds.length > 1 ? "s" : ""}`,
          description: "Review recent build errors before the next deploy attempt.",
          href: "/builds",
          cta: "Open builds",
          tone: "danger",
        }
      : null,
    unhealthyResources.length > 0
      ? {
          id: "resource-health",
          title: `${unhealthyResources.length} resources need attention`,
          description: "Stopped or errored containers usually block app access or data tasks.",
          href: "/projects",
          cta: "Inspect resources",
          tone: "warning",
        }
      : null,
    safeSources.length === 0
      ? {
          id: "missing-source",
          title: "No Git source connected",
          description: "Connect GitHub App or PAT before creating git-based app resources.",
          href: "/sources",
          cta: "Connect source",
          tone: "warning",
        }
      : null,
    baseDomains.length === 0
      ? {
          id: "missing-domain",
          title: "No base domain configured",
          description: "Configure base domains to publish HTTP resources with managed routing.",
          href: "/domains",
          cta: "Add domain",
          tone: "warning",
        }
      : null,
    settings?.app_domain
      ? null
      : {
          id: "missing-ingress",
          title: "App ingress is not configured",
          description: "Set the platform app domain so operators can reach the admin surface reliably.",
          href: "/settings",
          cta: "Open settings",
          tone: "neutral",
        },
  ].filter((item): item is AttentionItem => Boolean(item))

  const activityItems: ActivityItem[] = [
    ...buildItems.map((build) => ({
      id: `build-${build.id}`,
      title: build.image_tag,
      description: formatBuildLabel(build),
      timestamp: build.updated_at || build.created_at,
      href: "/builds",
      status: build.status,
      kind: "build" as const,
    })),
    ...resources.map((resource) => ({
      id: `resource-${resource.id}`,
      title: resource.name,
      description: `${resource.projectName} / ${resource.environmentName}`,
      timestamp: resource.updated_at || resource.created_at,
      href: `/resources/${resource.id}`,
      status: resource.status,
      kind: "resource" as const,
    })),
  ]
    .sort((left, right) => {
      const leftTime = new Date(left.timestamp).getTime()
      const rightTime = new Date(right.timestamp).getTime()
      return rightTime - leftTime
    })
    .slice(0, 6)

  const topResources = [...resources]
    .sort((left, right) => {
      const leftPriority =
        left.status === "error" ? 0 : left.status === "stopped" ? 1 : left.status === "building" ? 2 : 3
      const rightPriority =
        right.status === "error" ? 0 : right.status === "stopped" ? 1 : right.status === "building" ? 2 : 3

      if (leftPriority !== rightPriority) return leftPriority - rightPriority

      return new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime()
    })
    .slice(0, 5)

  const setupItems = getSetupStatus(settings, baseDomains, safeSources.length)
  const anyLoading =
    projectsLoading || buildsLoading || sourcesLoading || settingsLoading || domainsLoading

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<DashboardIcon className="size-5" />}
        title={t("common.dashboard")}
        description="Overview nhanh tình trạng platform, workloads và các việc cần xử lý."
        headerRight={
          <div className="flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={reconcileResources.isPending}
              onClick={() => {
                reconcileResources.mutate(undefined, {
                  onSuccess: (summary) => {
                    toast.success(t("common.runtimeSyncCompleted", summary))
                  },
                  onError: () => {
                    toast.error(t("common.runtimeSyncFailed"))
                  },
                })
              }}
            >
              {reconcileResources.isPending ? t("common.syncing") : t("common.syncRuntime")}
            </Button>
            <Button asChild size="sm" variant="outline">
              <Link to="/projects">Projects</Link>
            </Button>
            <Button asChild size="sm">
              <Link to="/sources">Connect Git</Link>
            </Button>
          </div>
        }
      />

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {anyLoading ? (
          Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-32 rounded-2xl" />
          ))
        ) : (
          <>
            <StatCard
              label="Projects"
              value={safeProjects.length}
              hint={`${environmentsCount} environments total`}
            />
            <StatCard
              label="Resources"
              value={resources.length}
              hint={`${runningResources.length} running right now`}
            />
            <StatCard
              label="Git-based apps"
              value={gitResources.length}
              hint={`${safeSources.length} source connection${safeSources.length === 1 ? "" : "s"}`}
            />
            <StatCard
              label="Attention"
              value={failedBuilds.length + unhealthyResources.length}
              hint={`${failedBuilds.length} failed builds, ${unhealthyResources.length} unhealthy resources`}
              tone={failedBuilds.length + unhealthyResources.length > 0 ? "danger" : "default"}
            />
          </>
        )}
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <SectionCard
          icon={<AlertTriangleIcon className="size-5" />}
          title="Needs Attention"
          description="Các việc nên xử lý trước để tránh block deploy hoặc routing."
          contentClassName="flex flex-col gap-3"
        >
          {anyLoading ? (
            <>
              <Skeleton className="h-24 rounded-2xl" />
              <Skeleton className="h-24 rounded-2xl" />
              <Skeleton className="h-24 rounded-2xl" />
            </>
          ) : attentionItems.length === 0 ? (
            <div className="rounded-2xl border border-border/70 bg-muted/20 p-6">
              <p className="font-medium">Không có cảnh báo nổi bật.</p>
              <p className="mt-1 text-sm text-muted-foreground">
                Build queue đang sạch và chưa thấy resource nào ở trạng thái lỗi hoặc dừng.
              </p>
            </div>
          ) : (
            attentionItems.map((item) => <AttentionCard key={item.id} item={item} />)
          )}
        </SectionCard>

        <SectionCard
          icon={<ActivityIcon className="size-5" />}
          title="Recent Activity"
          description="Biến động mới nhất từ builds và resources."
          headerRight={
            <Button asChild size="sm" variant="outline">
              <Link to="/builds">Open builds</Link>
            </Button>
          }
          contentClassName="flex flex-col gap-3"
        >
          {anyLoading ? (
            <>
              <Skeleton className="h-24 rounded-2xl" />
              <Skeleton className="h-24 rounded-2xl" />
              <Skeleton className="h-24 rounded-2xl" />
            </>
          ) : activityItems.length === 0 ? (
            <p className="text-sm text-muted-foreground">Chưa có activity để hiển thị.</p>
          ) : (
            activityItems.map((item) => <ActivityRow key={item.id} item={item} />)
          )}
        </SectionCard>
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <SectionCard
          icon={<ServerCogIcon className="size-5" />}
          title="Resources Snapshot"
          description="Những resource đáng chú ý nhất theo trạng thái và thời gian cập nhật."
          headerRight={
            <Button asChild size="sm" variant="outline">
              <Link to="/projects">Open projects</Link>
            </Button>
          }
          contentClassName="flex flex-col gap-3"
        >
          {projectsLoading ? (
            <>
              <Skeleton className="h-24 rounded-2xl" />
              <Skeleton className="h-24 rounded-2xl" />
              <Skeleton className="h-24 rounded-2xl" />
            </>
          ) : topResources.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-border px-4 py-8 text-sm text-muted-foreground">
              Chưa có resource nào. Tạo project và environment trước, rồi thêm app hoặc database.
            </div>
          ) : (
            topResources.map((resource) => (
              <ResourceRow key={resource.id} resource={resource} />
            ))
          )}
        </SectionCard>

        <div className="grid gap-6">
          <SectionCard
            icon={<PlayCircleIcon className="size-5" />}
            title="Quick Actions"
            description="Các lối đi ngắn cho workflow vận hành thường gặp."
            contentClassName="grid gap-3 sm:grid-cols-2"
          >
            <Button asChild variant="outline" className="justify-between">
              <Link to="/projects">
                New project
                <ArrowRightIcon data-icon="inline-end" />
              </Link>
            </Button>
            <Button asChild variant="outline" className="justify-between">
              <Link to="/builds">
                Trigger build
                <ArrowRightIcon data-icon="inline-end" />
              </Link>
            </Button>
            <Button asChild variant="outline" className="justify-between">
              <Link to="/sources">
                Connect GitHub
                <ArrowRightIcon data-icon="inline-end" />
              </Link>
            </Button>
            <Button asChild variant="outline" className="justify-between">
              <Link to="/domains">
                Add domain
                <ArrowRightIcon data-icon="inline-end" />
              </Link>
            </Button>
          </SectionCard>

          <SectionCard
            icon={<Settings2Icon className="size-5" />}
            title="Platform Setup"
            description="Checklist cấu hình nền tảng để publish app ổn định."
            headerRight={
              <Button asChild size="sm" variant="outline">
                <Link to="/settings">Settings</Link>
              </Button>
            }
            contentClassName="flex flex-col gap-3"
          >
            {settingsLoading || domainsLoading || sourcesLoading ? (
              <>
                <Skeleton className="h-20 rounded-2xl" />
                <Skeleton className="h-20 rounded-2xl" />
                <Skeleton className="h-20 rounded-2xl" />
              </>
            ) : (
              setupItems.map((item) => (
                <Link
                  key={item.label}
                  to={item.href}
                  className="flex items-center justify-between gap-4 rounded-2xl border border-border/70 bg-background/70 p-4 transition-colors hover:bg-muted/40"
                >
                  <div className="min-w-0">
                    <p className="font-medium">{item.label}</p>
                    <p className="mt-1 text-sm text-muted-foreground">{item.value}</p>
                  </div>
                  <Badge variant={item.done ? "default" : "secondary"}>
                    {item.done ? "Ready" : "Pending"}
                  </Badge>
                </Link>
              ))
            )}
          </SectionCard>

          <SectionCard
            icon={<GlobeIcon className="size-5" />}
            title="Routing Summary"
            description="Tóm tắt ingress và domain exposure của platform."
            contentClassName="grid gap-3 sm:grid-cols-2"
          >
            {settingsLoading || domainsLoading ? (
              <>
                <Skeleton className="h-20 rounded-2xl" />
                <Skeleton className="h-20 rounded-2xl" />
              </>
            ) : (
              <>
                <div className="rounded-2xl border border-border/70 bg-background/70 p-4">
                  <p className="text-sm text-muted-foreground">App ingress</p>
                  <p className="mt-2 font-medium">
                    {settings?.app_domain || "Not configured"}
                  </p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {settings?.app_domain
                      ? `${settings.app_tls_enabled ? "HTTPS" : "HTTP"} enabled`
                      : "Set this in Settings"}
                  </p>
                </div>
                <div className="rounded-2xl border border-border/70 bg-background/70 p-4">
                  <p className="text-sm text-muted-foreground">Base domains</p>
                  <p className="mt-2 font-medium">{baseDomains.length}</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {baseDomains.length > 0
                      ? `${baseDomains.filter((item) => item.wildcard_enabled).length} wildcard enabled`
                      : "Add at least one domain"}
                  </p>
                </div>
              </>
            )}
          </SectionCard>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-2xl border border-border/70 bg-card/70 p-4">
          <div className="flex items-center gap-2">
            <BoxIcon className="size-4 text-muted-foreground" />
            <p className="font-medium">Build queue</p>
          </div>
          <p className="mt-3 text-2xl font-semibold">{activeBuilds.length}</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Active build jobs đang chạy hoặc đang chờ.
          </p>
        </div>

        <div className="rounded-2xl border border-border/70 bg-card/70 p-4">
          <div className="flex items-center gap-2">
            <FolderGit2Icon className="size-4 text-muted-foreground" />
            <p className="font-medium">Last source activity</p>
          </div>
          <p className="mt-3 text-base font-semibold">
            {safeSources[0] ? formatDateTime(safeSources[0].updated_at) : "No source connected"}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Connection mới nhất trong danh sách nguồn mã.
          </p>
        </div>

        <div className="rounded-2xl border border-border/70 bg-card/70 p-4">
          <div className="flex items-center gap-2">
            <GlobeIcon className="size-4 text-muted-foreground" />
            <p className="font-medium">Public IP</p>
          </div>
          <p className="mt-3 text-base font-semibold">
            {settings?.public_ip || "Not configured"}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Dùng cho DNS, wildcard routing và expose services.
          </p>
        </div>
      </div>
    </div>
  )
}
