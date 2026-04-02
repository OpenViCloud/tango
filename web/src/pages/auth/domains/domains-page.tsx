import { toast } from "sonner"
import { PlusIcon, Settings2Icon, Trash2 } from "lucide-react"
import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { useGetSettings } from "@/hooks/api/use-settings"
import {
  useCreateBaseDomain,
  useDeleteBaseDomain,
  useGetBaseDomains,
} from "@/hooks/api/use-base-domains"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { appIcons } from "@/lib/icons"
import { useState } from "react"

const DomainsIcon = appIcons.domains

const DOMAIN_PATTERN =
  /^(?=.{1,253}$)(?!-)(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[A-Za-z]{2,}$/

function isValidDomain(value: string) {
  const normalized = value.trim().toLowerCase()
  if (!normalized) return false
  if (normalized.includes("://")) return false
  if (normalized.includes("/")) return false
  return DOMAIN_PATTERN.test(normalized)
}

export function DomainsPage() {
  const { t } = useTranslation()
  const { data: settings } = useGetSettings()
  const { data: baseDomains = [] } = useGetBaseDomains()
  const createBaseDomainMutation = useCreateBaseDomain()
  const deleteBaseDomainMutation = useDeleteBaseDomain()
  const [newBaseDomain, setNewBaseDomain] = useState("")
  const [newBaseDomainWildcard, setNewBaseDomainWildcard] = useState(true)
  const trimmedBaseDomain = newBaseDomain.trim()
  const baseDomainValid = isValidDomain(trimmedBaseDomain)
  const showBaseDomainError = trimmedBaseDomain.length > 0 && !baseDomainValid

  const handleAddBaseDomain = () => {
    const domain = trimmedBaseDomain.toLowerCase()
    if (!baseDomainValid) return
    createBaseDomainMutation.mutate(
      { domain, wildcard_enabled: newBaseDomainWildcard },
      {
        onSuccess: () => {
          setNewBaseDomain("")
          setNewBaseDomainWildcard(true)
          toast.success(t("settings.baseDomains.added"))
        },
        onError: () => toast.error(t("settings.baseDomains.addFailed")),
      }
    )
  }

  const handleDeleteBaseDomain = (id: string) => {
    deleteBaseDomainMutation.mutate(id, {
      onSuccess: () => toast.success(t("settings.baseDomains.deleted")),
      onError: () => toast.error(t("settings.baseDomains.deleteFailed")),
    })
  }

  const appURL =
    settings?.app_domain && settings.app_tls_enabled
      ? `https://${settings.app_domain}`
      : settings?.app_domain
        ? `http://${settings.app_domain}`
        : "Not configured"

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<DomainsIcon />}
        title={t("domains.page.title")}
        description={t("domains.page.description")}
      />

      <SectionCard>
        <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
          <div className="max-w-2xl space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="outline">{t("domains.overview.appIngress")}</Badge>
              <Badge variant={settings?.app_tls_enabled ? "default" : "secondary"}>
                {settings?.app_tls_enabled ? "HTTPS" : "HTTP"}
              </Badge>
            </div>
            <div>
              <h2 className="text-lg font-semibold">{t("domains.overview.title")}</h2>
              <p className="text-sm text-muted-foreground">{t("domains.overview.description")}</p>
            </div>
            <div className="rounded-2xl border border-border/70 bg-muted/30 p-4">
              <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                {t("domains.overview.appURL")}
              </p>
              <p className="mt-2 break-all font-mono text-sm text-foreground">{appURL}</p>
            </div>
          </div>

          <Button asChild variant="outline" className="shrink-0">
            <Link to="/settings">
              <Settings2Icon className="mr-2 size-4" />
              {t("domains.overview.manageSettings")}
            </Link>
          </Button>
        </div>
      </SectionCard>

      <SectionCard>
        <div className="flex flex-col gap-5">
          <div className="flex flex-col gap-1">
            <h2 className="text-lg font-semibold">{t("settings.baseDomains.title")}</h2>
            <p className="text-sm text-muted-foreground">{t("settings.baseDomains.description")}</p>
          </div>

          {baseDomains.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-border px-4 py-8 text-sm text-muted-foreground">
              {t("settings.baseDomains.empty")}
            </div>
          ) : (
            <div className="overflow-hidden rounded-2xl border border-border/70">
              <ul className="divide-y">
                {baseDomains.map((bd) => (
                  <li
                    key={bd.id}
                    className="flex flex-col gap-3 px-4 py-4 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div className="space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-mono text-sm text-foreground">{bd.domain}</span>
                        {bd.wildcard_enabled ? (
                          <Badge variant="default" className="text-xs">
                            {t("settings.baseDomains.wildcardLabel")}
                          </Badge>
                        ) : (
                          <Badge variant="secondary" className="text-xs">
                            {t("domains.baseDomains.manualLabel")}
                          </Badge>
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {bd.wildcard_enabled
                          ? t("domains.baseDomains.wildcardHint", { domain: bd.domain })
                          : t("domains.baseDomains.manualHint")}
                      </p>
                    </div>

                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 shrink-0 text-destructive hover:text-destructive"
                      disabled={deleteBaseDomainMutation.isPending}
                      onClick={() => handleDeleteBaseDomain(bd.id)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div className="rounded-2xl border border-border/70 bg-muted/20 p-4 sm:p-5">
            <div className="space-y-4">
              <div className="flex flex-col gap-1.5">
                <Label>{t("domains.add.domainLabel")}</Label>
                <Input
                  placeholder={t("settings.baseDomains.addPlaceholder")}
                  value={newBaseDomain}
                  onChange={(e) => setNewBaseDomain(e.target.value)}
                  disabled={createBaseDomainMutation.isPending}
                  aria-invalid={showBaseDomainError}
                />
                {showBaseDomainError ? (
                  <p className="text-xs text-destructive">
                    {t("domains.add.domainInvalid")}
                  </p>
                ) : (
                  <p className="text-xs text-muted-foreground">
                    {t("domains.add.domainHint")}
                  </p>
                )}
              </div>

              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="flex items-center justify-between rounded-xl border border-border/70 bg-background/80 px-3 py-3 sm:min-w-[320px] sm:justify-start sm:gap-4">
                  <div className="min-w-0">
                    <Label>{t("domains.add.wildcardLabel")}</Label>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {t("domains.add.wildcardHint")}
                    </p>
                  </div>
                  <Switch
                    checked={newBaseDomainWildcard}
                    onCheckedChange={setNewBaseDomainWildcard}
                    disabled={createBaseDomainMutation.isPending}
                  />
                </div>

                <Button
                  onClick={handleAddBaseDomain}
                  disabled={createBaseDomainMutation.isPending || !baseDomainValid}
                  className="sm:min-w-[160px]"
                >
                  <PlusIcon className="mr-2 size-4" />
                  {t("domains.add.action")}
                </Button>
              </div>
            </div>
          </div>
        </div>
      </SectionCard>
    </div>
  )
}
