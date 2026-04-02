import type { ResourceModel } from "@/@types/models"

import { ConfigSidebar } from "@/pages/auth/projects/components/ConfigSidebar"
import { ConfigGeneralForm, type EnvEntry, type PortEntry } from "@/pages/auth/resources/components/ConfigGeneralForm"
import { PersistentStorageForm, type VolumeEntry } from "@/pages/auth/resources/components/PersistentStorageForm"

type ResourceConfigurationTabProps = {
  resource: ResourceModel
  activeSection: string
  onSelectSection: (section: string) => void
  resourceName: string
  setResourceName: (name: string) => void
  portEntries: PortEntry[]
  setPortEntries: (ports: PortEntry[]) => void
  envEntries: EnvEntry[]
  setEnvEntries: (entries: EnvEntry[]) => void
  volumeEntries: VolumeEntry[]
  setVolumeEntries: (entries: VolumeEntry[]) => void
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
  resourceName,
  setResourceName,
  portEntries,
  setPortEntries,
  envEntries,
  setEnvEntries,
  volumeEntries,
  setVolumeEntries,
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
        className={`${sidebarOpen ? "translate-x-0" : "-translate-x-full"} fixed top-0 left-0 z-40 min-h-[calc(100vh-120px)] w-64 shrink-0 border-r border-border/80 bg-card/95 p-4 transition-transform md:sticky md:top-auto md:min-h-0 md:translate-x-0 md:bg-transparent md:transition-none`}
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
          className="fixed inset-0 z-30 bg-background/70 md:hidden"
          onClick={onDismissSidebar}
        />
      ) : null}

      <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
        {activeSection === "Persistent Storage" ? (
          <PersistentStorageForm
            entries={volumeEntries}
            setEntries={setVolumeEntries}
            onSave={onSave}
            pending={pending}
          />
        ) : (
          <ConfigGeneralForm
            resource={resource}
            resourceName={resourceName}
            setResourceName={setResourceName}
            portEntries={portEntries}
            setPortEntries={setPortEntries}
            envEntries={envEntries}
            setEnvEntries={setEnvEntries}
            onSave={onSave}
            pending={pending}
            isLoadingEnvVars={isLoadingEnvVars}
            isEnvVarsError={isEnvVarsError}
          />
        )}
      </main>
    </>
  )
}
