import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { actionIcons } from "@/lib/icons"

export type VolumeEntry = {
  source: string
  target: string
  mode: "rw" | "ro"
}

type PersistentStorageFormProps = {
  entries: VolumeEntry[]
  setEntries: (entries: VolumeEntry[]) => void
  onSave: () => void
  pending: boolean
}

const CreateIcon = actionIcons.create
const DeleteIcon = actionIcons.delete

export function PersistentStorageForm({
  entries,
  setEntries,
  onSave,
  pending,
}: PersistentStorageFormProps) {
  return (
    <div className="flex flex-col gap-8">
      <div className="flex items-center gap-3">
        <h2 className="text-xl font-bold text-foreground">Persistent Storage</h2>
        <Button size="sm" variant="outline" onClick={onSave} disabled={pending}>
          {pending ? "Saving..." : "Save"}
        </Button>
      </div>

      <div className="rounded-xl border border-border bg-card/40 p-4 text-sm text-muted-foreground">
        Source path is relative to the configured resource mount root. Target path must be an absolute path inside the container.
      </div>

      <div className="flex flex-col gap-3">
        {entries.map((entry, index) => (
          <div key={index} className="grid gap-3 md:grid-cols-[1fr_1fr_160px_auto]">
            <div className="flex flex-col gap-1.5">
              <Label>Source Path</Label>
              <Input
                value={entry.source}
                onChange={(e) => {
                  const next = [...entries]
                  next[index] = { ...next[index], source: e.target.value }
                  setEntries(next)
                }}
                placeholder="e.g. databasus-data"
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Container Path</Label>
              <Input
                value={entry.target}
                onChange={(e) => {
                  const next = [...entries]
                  next[index] = { ...next[index], target: e.target.value }
                  setEntries(next)
                }}
                placeholder="e.g. /databasus-data"
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Mode</Label>
              <Select
                value={entry.mode}
                onValueChange={(value: "rw" | "ro") => {
                  const next = [...entries]
                  next[index] = { ...next[index], mode: value }
                  setEntries(next)
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="rw">Read / Write</SelectItem>
                  <SelectItem value="ro">Read Only</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-end">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() =>
                  entries.length > 1
                    ? setEntries(entries.filter((_, i) => i !== index))
                    : setEntries([{ source: "", target: "", mode: "rw" }])
                }
              >
                <DeleteIcon />
              </Button>
            </div>
          </div>
        ))}
      </div>

      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            setEntries([...entries, { source: "", target: "", mode: "rw" }])
          }
        >
          <CreateIcon data-icon="inline-start" />
          Add Mount
        </Button>
      </div>
    </div>
  )
}
