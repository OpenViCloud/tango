import { DatabaseIcon, GlobeIcon, NetworkIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import type { ResourceModel } from "@/@types/models"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { useGetResourceConnectionInfo } from "@/hooks/api/use-project"

type Props = {
  resource: ResourceModel
}

export function ResourceConnectionTab({ resource }: Props) {
  const { t } = useTranslation()
  const { data, isLoading } = useGetResourceConnectionInfo(resource.id)

  return (
    <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-col gap-4">
        <SectionCard>a
          <div className="flex flex-col gap-3">
            <div className="flex items-center gap-2">
              <DatabaseIcon className="h-4 w-4 text-muted-foreground" />
              <h2 className="text-base font-semibold">
                {t("projects.resource.connection.title")}
              </h2>
            </div>
            <p className="text-sm text-muted-foreground">
              {t("projects.resource.connection.description")}
            </p>
            <div className="rounded-2xl border border-border/70 bg-muted/20 p-4">
              <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                {t("projects.resource.connection.internalHost")}
              </p>
              <p className="mt-2 font-mono text-sm text-foreground">
                {data?.internal_host || (isLoading ? t("projects.resource.connection.loading") : resource.name)}
              </p>
            </div>
          </div>
        </SectionCard>

        <div className="grid gap-4 xl:grid-cols-2">
          {(data?.ports || []).map((port) => (
            <SectionCard key={port.id || `${port.internal_port}-${port.host_port}`}>
              <div className="flex flex-col gap-4">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant="outline" className="font-mono">
                    {t("projects.resource.connection.portBadge", {
                      port: port.internal_port,
                    })}
                  </Badge>
                  {port.label ? (
                    <Badge variant="secondary">{port.label}</Badge>
                  ) : null}
                </div>

                <div className="rounded-2xl border border-border/70 bg-background/70 p-4">
                  <div className="mb-2 flex items-center gap-2">
                    <NetworkIcon className="h-4 w-4 text-muted-foreground" />
                    <p className="text-sm font-medium">
                      {t("projects.resource.connection.internal")}
                    </p>
                  </div>
                  <p className="font-mono text-sm text-foreground">
                    {port.internal_endpoint}
                  </p>
                  <p className="mt-2 text-xs text-muted-foreground">
                    {t("projects.resource.connection.internalHint")}
                  </p>
                </div>

                <div className="rounded-2xl border border-border/70 bg-background/70 p-4">
                  <div className="mb-2 flex items-center gap-2">
                    <GlobeIcon className="h-4 w-4 text-muted-foreground" />
                    <p className="text-sm font-medium">
                      {t("projects.resource.connection.external")}
                    </p>
                  </div>
                  {port.external_endpoint ? (
                    <>
                      <p className="font-mono text-sm text-foreground">
                        {port.external_endpoint}
                      </p>
                      <p className="mt-2 text-xs text-muted-foreground">
                        {t("projects.resource.connection.externalHint")}
                      </p>
                    </>
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      {t("projects.resource.connection.notExposed")}
                    </p>
                  )}
                </div>
              </div>
            </SectionCard>
          ))}
        </div>
      </div>
    </main>
  )
}
