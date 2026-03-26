import type { ResourceModel } from "@/@types/models"

type ResourceBackupsTabProps = {
  resource: ResourceModel
}

export function ResourceBackupsTab({ resource }: ResourceBackupsTabProps) {
  return (
    <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
      <div className="rounded-2xl border bg-card p-6">
        <h2 className="text-xl font-semibold">Backups</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Backup history and restore actions for <span className="font-medium text-foreground">{resource.name}</span> will be shown here.
        </p>
      </div>
    </main>
  )
}
