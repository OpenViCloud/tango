import type { ResourceModel } from "@/@types/models"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useGetResourceLogs } from "@/hooks/api/use-project"
import { useState } from "react"

type ResourceLogsTabProps = {
  resource: ResourceModel
}

export function ResourceLogsTab({ resource }: ResourceLogsTabProps) {
  const { data, isLoading, isError } = useGetResourceLogs(resource.id, 200)
  const [pretty, setPretty] = useState(true)
  const lines = data?.lines ?? []
  const displayText = pretty ? lines.map(formatLogLine).join("\n") : lines.join("\n")

  return (
    <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
      <div className="rounded-2xl border bg-card p-6">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="text-xl font-semibold">Logs</h2>
            <p className="mt-2 text-sm text-muted-foreground">
              Runtime logs for <span className="font-medium text-foreground">{resource.name}</span>. Refreshes every 5 seconds.
            </p>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={() => setPretty((value) => !value)}>
            {pretty ? "Raw" : "Pretty"}
          </Button>
        </div>

        <div className="mt-4 rounded-xl border bg-muted/30 p-4">
          {isLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-4/5" />
              <Skeleton className="h-4 w-3/5" />
            </div>
          ) : isError ? (
            <p className="text-sm text-destructive">Could not load logs.</p>
          ) : lines.length === 0 ? (
            <p className="text-sm text-muted-foreground">No logs available.</p>
          ) : (
            <pre className="max-h-[520px] overflow-auto whitespace-pre-wrap break-all font-mono text-xs leading-relaxed">
              {displayText}
            </pre>
          )}
        </div>
      </div>
    </main>
  )
}

function formatLogLine(line: string) {
  try {
    const parsed = JSON.parse(line) as {
      t?: { $date?: string }
      s?: string
      c?: string
      ctx?: string
      msg?: string
      attr?: Record<string, unknown>
    }

    if (!parsed || typeof parsed !== "object") {
      return line
    }

    const timestamp = parsed.t?.$date ? new Date(parsed.t.$date).toLocaleString() : null
    const level = mongoLevelToText(parsed.s)
    const category = typeof parsed.c === "string" ? parsed.c : null
    const ctx = typeof parsed.ctx === "string" ? parsed.ctx : null
    const msg = typeof parsed.msg === "string" ? parsed.msg : line
    const attrs = parsed.attr && typeof parsed.attr === "object"
      ? Object.entries(parsed.attr)
          .map(([key, value]) => `${key}=${formatAttrValue(value)}`)
          .join(" ")
      : ""

    const meta = [timestamp, level, category, ctx].filter(Boolean).join(" ")
    return attrs ? `${meta} - ${msg}\n${attrs}` : `${meta} - ${msg}`
  } catch {
    return line
  }
}

function mongoLevelToText(value: string | undefined) {
  switch (value) {
    case "E":
      return "ERROR"
    case "W":
      return "WARN"
    case "I":
      return "INFO"
    case "D":
      return "DEBUG"
    default:
      return value ?? "LOG"
  }
}

function formatAttrValue(value: unknown): string {
  if (value == null) return "null"
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return String(value)
  }
  return JSON.stringify(value)
}
