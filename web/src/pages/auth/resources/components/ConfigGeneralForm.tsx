import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import type { ResourceModel } from "@/@types/models"
import { actionIcons } from "@/lib/icons"
import { useTranslation } from "react-i18next"

type EnvEntry = {
  key: string
  value: string
  is_secret: boolean
}

type ConfigGeneralFormProps = {
  resource: ResourceModel | null
  envEntries: EnvEntry[]
  setEnvEntries: (entries: EnvEntry[]) => void
  onSave: () => void
  pending: boolean
  isLoadingEnvVars: boolean
  isEnvVarsError: boolean
}

const CreateIcon = actionIcons.create
const DeleteIcon = actionIcons.delete

export function ConfigGeneralForm({
  resource,
  envEntries,
  setEnvEntries,
  onSave,
  pending,
  isLoadingEnvVars,
  isEnvVarsError,
}: ConfigGeneralFormProps) {
  const { t } = useTranslation()

  return (
    <div className="flex flex-col gap-8">
      <div className="flex items-center gap-3">
        <h2 className="text-xl font-bold text-foreground">
          {t("projects.resource.infoTitle")}
        </h2>
        <Button size="sm" variant="outline" onClick={onSave} disabled={pending}>
          {pending ? t("projects.resource.saving") : t("projects.resource.save")}
        </Button>
      </div>

      {resource ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <ReadOnlyField label={t("projects.nameLabel")} value={resource.name} />
          <ReadOnlyField
            label={t("projects.resource.typeLabel")}
            value={resource.type}
          />
          <ReadOnlyField
            label={t("projects.resource.statusLabel")}
            value={resource.status}
          />
          <ReadOnlyField label="Image" value={`${resource.image}:${resource.tag}`} />
          <ReadOnlyField
            label="Container ID"
            value={resource.container_id || "-"}
          />
          <ReadOnlyField
            label={t("projects.resource.portsLabel")}
            value={
              resource.ports.length > 0
                ? resource.ports
                    .map(
                      (port) =>
                        `${port.host_port}:${port.internal_port}/${port.proto}`
                    )
                    .join(", ")
                : "-"
            }
          />
        </div>
      ) : null}

      <div className="flex flex-col gap-4">
        <h3 className="text-lg font-bold text-foreground">
          {t("projects.resource.envTitle")}
        </h3>

        {isLoadingEnvVars ? (
          <div className="flex flex-col gap-3">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : isEnvVarsError ? (
          <div className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
            {t("projects.resource.loadEnvFailed")}
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            {envEntries.map((entry, index) => (
              <div
                key={index}
                className="grid gap-3 md:grid-cols-[1fr_1fr_auto]"
              >
                <div className="flex flex-col gap-1.5">
                  <Label>{t("docker.container.envKey")}</Label>
                  <Input
                    value={entry.key}
                    onChange={(e) => {
                      const next = [...envEntries]
                      next[index] = { ...next[index], key: e.target.value }
                      setEnvEntries(next)
                    }}
                    placeholder={t("docker.container.envKey")}
                  />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label>{t("docker.container.envValue")}</Label>
                  <Input
                    value={entry.value}
                    onChange={(e) => {
                      const next = [...envEntries]
                      next[index] = { ...next[index], value: e.target.value }
                      setEnvEntries(next)
                    }}
                    placeholder={t("docker.container.envValue")}
                  />
                </div>
                <div className="flex items-end">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={() =>
                      envEntries.length > 1
                        ? setEnvEntries(
                            envEntries.filter(
                              (_item, itemIndex) => itemIndex !== index
                            )
                          )
                        : setEnvEntries([
                            { key: "", value: "", is_secret: false },
                          ])
                    }
                  >
                    <DeleteIcon />
                  </Button>
                </div>
              </div>
            ))}

            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() =>
                  setEnvEntries([
                    ...envEntries,
                    { key: "", value: "", is_secret: false },
                  ])
                }
              >
                <CreateIcon data-icon="inline-start" />
                {t("docker.container.addEnv")}
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      <Input value={value} readOnly />
    </div>
  )
}

export type { EnvEntry, ConfigGeneralFormProps }
