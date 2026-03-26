import type { ResourceModel } from "@/@types/models"

type ResourceTerminalTabProps = {
  resource: ResourceModel
}

export function ResourceTerminalTab({ resource }: ResourceTerminalTabProps) {
  return (
    <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
      <div className="rounded-2xl border bg-card p-6">
        <h2 className="text-xl font-semibold">Terminal</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Interactive terminal support for <span className="font-medium text-foreground">{resource.name}</span> can be attached here.
        </p>
      </div>
    </main>
  )
}
