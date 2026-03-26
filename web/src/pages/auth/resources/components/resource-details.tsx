import { useState } from "react"
import { ChevronRight, Menu, Play, Square, X } from "lucide-react"
import { useNavigate } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import type { ResourceModel } from "@/@types/models"
import { Button } from "@/components/ui/button"
import { type EnvEntry } from "./ConfigGeneralForm"
import { ResourceBackupsTab } from "./tabs/ResourceBackupsTab"
import { ResourceConfigurationTab } from "./tabs/ResourceConfigurationTab"
import { ResourceLogsTab } from "./tabs/ResourceLogsTab"
import { ResourceTerminalTab } from "./tabs/ResourceTerminalTab"

type ResourceDetailsProps = {
  resource: ResourceModel
  initialEnvEntries: EnvEntry[]
  onSave: (entries: EnvEntry[]) => void
  onStart: () => void
  onStop: () => void
  pending: boolean
  actionPending: boolean
  isLoadingEnvVars: boolean
  isEnvVarsError: boolean
}

const tabs = ["Configuration", "Logs", "Terminal", "Backups"]

export default function ResourceDetails({
  resource,
  initialEnvEntries,
  onSave,
  onStart,
  onStop,
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

  const statusDotClass =
    resource.status === "running" ? "bg-green-500" : "bg-destructive"
  const statusTextClass =
    resource.status === "running" ? "text-green-600" : "text-destructive"
  const isRunning = resource.status === "running"

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="border-b border-border px-4 py-4 sm:px-6">
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
          </span>
        </div>
      </div>

      <div className="flex items-center justify-between border-b border-border px-4 sm:px-6">
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
        {isRunning ? (
          <Button
            type="button"
            size="sm"
            disabled={actionPending}
            onClick={onStop}
            className="ml-4 shrink-0 gap-2 border border-border bg-transparent text-foreground hover:bg-secondary"
          >
            <Square className="h-4 w-4" />
            {t("projects.resource.stop")}
          </Button>
        ) : (
          <Button
            type="button"
            size="sm"
            disabled={actionPending}
            onClick={onStart}
            className="ml-4 shrink-0 gap-2 border border-border bg-transparent text-foreground hover:bg-secondary"
          >
            <Play className="h-4 w-4" />
            {t("projects.resource.start")}
          </Button>
        )}
      </div>

      <div className="relative flex">
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
              envEntries={envEntries}
              setEnvEntries={setEnvEntries}
              onSave={() => onSave(envEntries)}
              pending={pending}
              isLoadingEnvVars={isLoadingEnvVars}
              isEnvVarsError={isEnvVarsError}
              sidebarOpen={sidebarOpen}
              onDismissSidebar={() => setSidebarOpen(false)}
            />
          </>
        ) : null}

        {activeTab === "Logs" ? <ResourceLogsTab resource={resource} /> : null}
        {activeTab === "Terminal" ? (
          <ResourceTerminalTab
            key={`${resource.id}:${resource.status}:${resource.container_id ?? ""}`}
            resource={resource}
          />
        ) : null}
        {activeTab === "Backups" ? (
          <ResourceBackupsTab resource={resource} />
        ) : null}
      </div>
    </div>
  )
}
