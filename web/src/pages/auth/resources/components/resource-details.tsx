import { useState } from "react"
import { ChevronRight, Hammer, Menu, Play, Square, X, ChevronsUpDown } from "lucide-react"
import { Input } from "@/components/ui/input"
import { useNavigate } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import type { ResourceModel } from "@/@types/models"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { type EnvEntry, type PortEntry } from "./ConfigGeneralForm"
import { type VolumeEntry } from "./PersistentStorageForm"
import { ResourceBackupsTab } from "./tabs/ResourceBackupsTab"
import { ResourceConnectionTab } from "./tabs/ResourceConnectionTab"
import { ResourceConfigurationTab } from "./tabs/ResourceConfigurationTab"
import { ResourceDomainsTab } from "./tabs/ResourceDomainsTab"
import { ResourceLogsTab } from "./tabs/ResourceLogsTab"
import { ResourceTerminalTab } from "./tabs/ResourceTerminalTab"

type ResourceDetailsProps = {
  resource: ResourceModel
  initialEnvEntries: EnvEntry[]
  onSave: (
    entries: EnvEntry[],
    name: string,
    ports: PortEntry[],
    volumes: VolumeEntry[]
  ) => void
  onStart: () => void
  onStop: () => void
  onBuild?: () => void
  onScale?: (replicas: number) => void
  isSwarmManager?: boolean
  scalePending?: boolean
  pending: boolean
  actionPending: boolean
  isLoadingEnvVars: boolean
  isEnvVarsError: boolean
}

export default function ResourceDetails({
  resource,
  initialEnvEntries,
  onSave,
  onStart,
  onStop,
  onBuild,
  onScale,
  isSwarmManager = false,
  scalePending = false,
  pending,
  actionPending,
  isLoadingEnvVars,
  isEnvVarsError,
}: ResourceDetailsProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [activeSection, setActiveSection] = useState("General")
  const [activeTab, setActiveTab] = useState("Configuration")
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [envEntries, setEnvEntries] = useState<EnvEntry[]>(initialEnvEntries)
  const [resourceName, setResourceName] = useState(resource.name)
  const [portEntries, setPortEntries] = useState<PortEntry[]>(
    resource.ports.length > 0
      ? resource.ports.map((p) => ({
          host_port: String(p.host_port),
          internal_port: String(p.internal_port),
          proto: p.proto || "tcp",
          label: p.label || "",
        }))
      : [{ host_port: "", internal_port: "", proto: "tcp", label: "" }]
  )
  const [volumeEntries, setVolumeEntries] = useState<VolumeEntry[]>(() => {
    const rawVolumes = Array.isArray(resource.config?.volumes)
      ? resource.config.volumes
      : []
    const parsed = rawVolumes
      .map((item) => {
        if (typeof item !== "string") return null
        const parts = item.split(":")
        if (parts.length < 2 || parts.length > 3) return null
        return {
          source: parts[0] ?? "",
          target: parts[1] ?? "",
          mode: parts[2] === "ro" ? "ro" : "rw",
        } satisfies VolumeEntry
      })
      .filter((item): item is VolumeEntry => item !== null)

    return parsed.length > 0
      ? parsed
      : [{ source: "", target: "", mode: "rw" }]
  })

  const [scaleReplicas, setScaleReplicas] = useState(
    Math.max(1, resource.replicas ?? 1)
  )

  const statusDotClass =
    resource.status === "running" ? "bg-green-500" : "bg-destructive"
  const statusTextClass =
    resource.status === "running" ? "text-green-600" : "text-destructive"
  const isRunning = resource.status === "running"
  const tabs = [
    "Configuration",
    ...(resource.type === "db" ? ["Connection"] : []),
    "Logs",
    "Terminal",
    "Domains",
    "Backups",
  ]

  return (
    <div className="flex flex-col gap-6 text-foreground">
      <Card className="overflow-hidden border-none bg-card shadow-panel">
        <div className="border-b border-border/80 px-4 py-5 sm:px-6">
          <h1 className="text-2xl font-bold sm:text-3xl">
            {t("projects.resource.editPageTitle")}
          </h1>
          <div className="mt-1 flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
            <button
              type="button"
              onClick={() => navigate({ to: "/projects" })}
              className="cursor-pointer hover:text-foreground"
            >
              {t("projects.page.title")}
            </button>
            <ChevronRight className="h-3.5 w-3.5 shrink-0" />
            <span className="break-all text-foreground">{resource.name}</span>
            <ChevronRight className="h-3.5 w-3.5 shrink-0" />
            <span className="flex items-center gap-1.5">
              <span className={`inline-block h-2.5 w-2.5 rounded-full ${statusDotClass}`} />
              <span className={statusTextClass}>{resource.status}</span>
              {isSwarmManager && (resource.replicas ?? 1) > 0 && (
                <span className="ml-1 rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                  {resource.replicas ?? 1}×
                </span>
              )}
            </span>
          </div>
        </div>

        <div className="flex items-center justify-between border-b border-border/80 px-4 sm:px-6">
          <div className="flex gap-4 overflow-x-auto sm:gap-6">
            {tabs.map((tab) => (
              <button
                key={tab}
                type="button"
                onClick={() => setActiveTab(tab)}
                className={`border-b-2 py-3 text-sm whitespace-nowrap transition-colors ${
                  activeTab === tab
                    ? "border-accent text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                }`}
              >
                {tab}
              </button>
            ))}
          </div>
          <div className="ml-4 flex shrink-0 items-center gap-2">
            {resource.source_type === "git" && onBuild && (
              <Button
                type="button"
                size="sm"
                disabled={actionPending || resource.status === "building"}
                onClick={onBuild}
                className="gap-2 border border-border bg-transparent text-foreground hover:bg-secondary"
              >
                <Hammer className="h-4 w-4" />
                {resource.status === "building" ? "Building…" : "Build"}
              </Button>
            )}
            {/* Scale control — only visible in swarm mode */}
            {isSwarmManager && isRunning && onScale && (
              <div className="flex items-center gap-1">
                <Input
                  type="number"
                  min={1}
                  className="h-8 w-16 px-2 text-center text-sm"
                  value={scaleReplicas}
                  onChange={(e) =>
                    setScaleReplicas(Math.max(1, parseInt(e.target.value, 10) || 1))
                  }
                  disabled={scalePending}
                  title="Replicas"
                />
                <Button
                  type="button"
                  size="sm"
                  disabled={scalePending || scaleReplicas === (resource.replicas ?? 1)}
                  onClick={() => onScale(scaleReplicas)}
                  className="gap-1 border border-border bg-transparent text-foreground hover:bg-secondary"
                  title="Scale replicas"
                >
                  <ChevronsUpDown className="h-4 w-4" />
                  {scalePending ? "…" : "Scale"}
                </Button>
              </div>
            )}
            {isRunning ? (
              <Button
                type="button"
                size="sm"
                disabled={actionPending}
                onClick={onStop}
                className="gap-2 border border-border bg-transparent text-foreground hover:bg-secondary"
              >
                <Square className="h-4 w-4" />
                {t("projects.resource.stop")}
              </Button>
            ) : (
              <Button
                type="button"
                size="sm"
                disabled={actionPending || resource.status === "building" || resource.status === "created"}
                onClick={onStart}
                className="gap-2 border border-border bg-transparent text-foreground hover:bg-secondary"
              >
                <Play className="h-4 w-4" />
                {t("projects.resource.start")}
              </Button>
            )}
          </div>
        </div>

        <div className="relative flex bg-card/70">
          {activeTab === "Configuration" ? (
            <>
              <button
                type="button"
                onClick={() => setSidebarOpen(!sidebarOpen)}
                className="fixed right-4 bottom-4 z-50 rounded-full bg-accent p-3 text-accent-foreground shadow-lg md:hidden"
              >
                {sidebarOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
              </button>
              <ResourceConfigurationTab
                resource={resource}
                activeSection={activeSection}
                onSelectSection={setActiveSection}
                resourceName={resourceName}
                setResourceName={setResourceName}
                portEntries={portEntries}
                setPortEntries={setPortEntries}
                envEntries={envEntries}
                setEnvEntries={setEnvEntries}
                volumeEntries={volumeEntries}
                setVolumeEntries={setVolumeEntries}
                onSave={() =>
                  onSave(envEntries, resourceName, portEntries, volumeEntries)
                }
                pending={pending}
                isLoadingEnvVars={isLoadingEnvVars}
                isEnvVarsError={isEnvVarsError}
                sidebarOpen={sidebarOpen}
                onDismissSidebar={() => setSidebarOpen(false)}
              />
            </>
          ) : null}

          {activeTab === "Logs" ? <ResourceLogsTab resource={resource} /> : null}
          {activeTab === "Connection" ? (
            <ResourceConnectionTab resource={resource} />
          ) : null}
          {activeTab === "Terminal" ? (
            <ResourceTerminalTab
              key={`${resource.id}:${resource.status}:${resource.container_id ?? ""}`}
              resource={resource}
            />
          ) : null}
          {activeTab === "Domains" ? (
            <ResourceDomainsTab resource={resource} />
          ) : null}
          {activeTab === "Backups" ? (
            <ResourceBackupsTab resource={resource} />
          ) : null}
        </div>
      </Card>
    </div>
  )
}
