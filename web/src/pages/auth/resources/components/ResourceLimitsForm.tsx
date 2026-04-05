import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

// ── Conversion helpers ────────────────────────────────────────────────────────

const GB = 1024 * 1024 * 1024
const MB = 1024 * 1024
const CPU_NANO = 1_000_000_000

/** bytes → display string, e.g. 3221225472 → "3" (GB) */
function bytesToDisplay(bytes: number): { value: string; unit: "GB" | "MB" } {
  if (bytes === 0) return { value: "", unit: "GB" }
  if (bytes % GB === 0) return { value: String(bytes / GB), unit: "GB" }
  return { value: String(bytes / MB), unit: "MB" }
}

/** nanoCPUs → display string, e.g. 500000000 → "0.5" */
function nanoCPUsToDisplay(nano: number): string {
  if (nano === 0) return ""
  return String(nano / CPU_NANO)
}

function displayToBytes(value: string, unit: "GB" | "MB"): number {
  const num = parseFloat(value)
  if (!value || isNaN(num) || num <= 0) return 0
  return Math.round(num * (unit === "GB" ? GB : MB))
}

function displayToNanoCPUs(value: string): number {
  const num = parseFloat(value)
  if (!value || isNaN(num) || num <= 0) return 0
  return Math.round(num * CPU_NANO)
}

// ── Component ─────────────────────────────────────────────────────────────────

type ResourceLimitsFormProps = {
  memoryLimit: number  // bytes
  cpuLimit: number     // nanoCPUs
  onMemoryLimitChange: (bytes: number) => void
  onCPULimitChange: (nano: number) => void
  onSave: () => void
  pending: boolean
}

export function ResourceLimitsForm({
  memoryLimit,
  cpuLimit,
  onMemoryLimitChange,
  onCPULimitChange,
  onSave,
  pending,
}: ResourceLimitsFormProps) {
  const initialMem = bytesToDisplay(memoryLimit)
  const [memValue, setMemValue] = useState(initialMem.value)
  const [memUnit, setMemUnit] = useState<"GB" | "MB">(initialMem.unit)
  const [cpuValue, setCpuValue] = useState(nanoCPUsToDisplay(cpuLimit))

  function handleMemoryChange(value: string, unit: "GB" | "MB") {
    setMemValue(value)
    setMemUnit(unit)
    onMemoryLimitChange(displayToBytes(value, unit))
  }

  function handleCPUChange(value: string) {
    setCpuValue(value)
    onCPULimitChange(displayToNanoCPUs(value))
  }

  return (
    <div className="flex flex-col gap-8">
      <div className="flex items-center gap-3">
        <h2 className="text-xl font-bold text-foreground">Resource Limits</h2>
        <Button size="sm" variant="outline" onClick={onSave} disabled={pending}>
          {pending ? "Saving…" : "Save"}
        </Button>
      </div>

      <p className="text-sm text-muted-foreground">
        Hard limits applied to each container instance. Leave blank for unlimited.
      </p>

      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        {/* Memory */}
        <div className="flex flex-col gap-2">
          <Label>Memory Limit</Label>
          <div className="flex gap-2">
            <Input
              type="number"
              min={0}
              step={memUnit === "GB" ? 0.5 : 128}
              placeholder="unlimited"
              value={memValue}
              onChange={(e) => handleMemoryChange(e.target.value, memUnit)}
              className="flex-1"
            />
            <select
              value={memUnit}
              onChange={(e) => handleMemoryChange(memValue, e.target.value as "GB" | "MB")}
              className="rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="GB">GB</option>
              <option value="MB">MB</option>
            </select>
          </div>
          <p className="text-xs text-muted-foreground">
            {memoryLimit > 0
              ? `= ${memoryLimit.toLocaleString()} bytes`
              : "No limit — container can use all available memory"}
          </p>
        </div>

        {/* CPU */}
        <div className="flex flex-col gap-2">
          <Label>CPU Limit</Label>
          <Input
            type="number"
            min={0}
            step={0.25}
            placeholder="unlimited"
            value={cpuValue}
            onChange={(e) => handleCPUChange(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            {cpuLimit > 0
              ? `= ${cpuLimit.toLocaleString()} nanoCPUs`
              : "No limit — container can use all available CPU"}
          </p>
          <div className="flex flex-wrap gap-1.5">
            {[0.25, 0.5, 1, 2, 4].map((v) => (
              <button
                key={v}
                type="button"
                onClick={() => handleCPUChange(String(v))}
                className="rounded border border-border px-2 py-0.5 text-xs text-muted-foreground hover:border-accent hover:text-accent"
              >
                {v}×
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
