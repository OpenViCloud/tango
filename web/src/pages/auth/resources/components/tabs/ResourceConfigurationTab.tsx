import type { ResourceModel } from "@/@types/models"

import { ConfigSidebar } from "@/pages/auth/projects/components/ConfigSidebar"
import { ConfigGeneralForm, type EnvEntry } from "@/pages/auth/resources/components/ConfigGeneralForm"

type ResourceConfigurationTabProps = {
  resource: ResourceModel
  activeSection: string
  onSelectSection: (section: string) => void
  envEntries: EnvEntry[]
  setEnvEntries: (entries: EnvEntry[]) => void
  onSave: () => void
  pending: boolean
  isLoadingEnvVars: boolean
  isEnvVarsError: boolean
  sidebarOpen: boolean
  onDismissSidebar: () => void
}

export function ResourceConfigurationTab({
  resource,
  activeSection,
  onSelectSection,
  envEntries,
  setEnvEntries,
  onSave,
  pending,
  isLoadingEnvVars,
  isEnvVarsError,
  sidebarOpen,
  onDismissSidebar,
}: ResourceConfigurationTabProps) {
  return (
    <>
      <aside
        className={`${sidebarOpen ? "translate-x-0" : "-translate-x-full"} fixed top-0 left-0 z-40 min-h-[calc(100vh-120px)] w-64 shrink-0 border-r border-border bg-background p-4 transition-transform md:sticky md:top-auto md:min-h-0 md:translate-x-0 md:transition-none`}
      >
        <ConfigSidebar
          active={activeSection}
          onSelect={(item) => {
            onSelectSection(item)
            onDismissSidebar()
          }}
        />
      </aside>

      {sidebarOpen ? (
        <div
          className="fixed inset-0 z-30 bg-background/60 md:hidden"
          onClick={onDismissSidebar}
        />
      ) : null}

      <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
        <ConfigGeneralForm
          resource={resource}
          envEntries={envEntries}
          setEnvEntries={setEnvEntries}
          onSave={onSave}
          pending={pending}
          isLoadingEnvVars={isLoadingEnvVars}
          isEnvVarsError={isEnvVarsError}
        />
      </main>
    </>
  )
}
