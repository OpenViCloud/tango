import type { ResourceModel } from "@/@types/models"

type ResourceLogsTabProps = {
  resource: ResourceModel
}

export function ResourceLogsTab({ resource }: ResourceLogsTabProps) {
  return (
    <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
      <div className="rounded-2xl border bg-card p-6">
        <h2 className="text-xl font-semibold">Logs</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Runtime logs for <span className="font-medium text-foreground">{resource.name}</span> will live here.
        </p>
      </div>
    </main>
  )
}
