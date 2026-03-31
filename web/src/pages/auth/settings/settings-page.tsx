import { useEffect, useState } from "react"
import { toast } from "sonner"
import { GlobeIcon, Settings2Icon } from "lucide-react"
import { Link } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { useGetSettings, useUpdateSettings } from "@/hooks/api/use-settings"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"

export function SettingsPage() {
  const { t } = useTranslation()
  const { data: settings, isLoading } = useGetSettings()
  const updateMutation = useUpdateSettings()

  const [publicIp, setPublicIp] = useState("")
  const [traefikNetwork, setTraefikNetwork] = useState("")
  const [certResolver, setCertResolver] = useState("letsencrypt")
  const [appDomain, setAppDomain] = useState("")
  const [appTLSEnabled, setAppTLSEnabled] = useState(false)
  const [appBackendURL, setAppBackendURL] = useState("http://app:8080")

  useEffect(() => {
    if (settings) {
      setPublicIp(settings.public_ip ?? "")
      setTraefikNetwork(settings.traefik_network ?? "")
      setCertResolver(settings.cert_resolver ?? "letsencrypt")
      setAppDomain(settings.app_domain ?? "")
      setAppTLSEnabled(settings.app_tls_enabled ?? false)
      setAppBackendURL(settings.app_backend_url ?? "http://app:8080")
    }
  }, [settings])

  const handleSave = () => {
    updateMutation.mutate(
      {
        public_ip: publicIp,
        traefik_network: traefikNetwork,
        cert_resolver: certResolver,
        app_domain: appDomain,
        app_tls_enabled: appTLSEnabled,
        app_backend_url: appBackendURL,
      },
      {
        onSuccess: () => toast.success(t("settings.saved")),
        onError: () => toast.error(t("settings.saveFailed")),
      }
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<Settings2Icon className="size-6" />}
        title={t("settings.page.title")}
        description={t("settings.page.description")}
      />

      <SectionCard>
        <div className="max-w-lg space-y-6">
          <div className="flex flex-col gap-1.5">
            <Label>{t("settings.publicIp.label")}</Label>
            <Input
              placeholder="115.74.113.215"
              value={publicIp}
              onChange={(e) => setPublicIp(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-muted-foreground">
              {t("settings.publicIp.hint")}
            </p>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>{t("settings.traefikNetwork.label")}</Label>
            <Input
              placeholder="bridge"
              value={traefikNetwork}
              onChange={(e) => setTraefikNetwork(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-muted-foreground">
              {t("settings.traefikNetwork.hint")}
            </p>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>{t("settings.certResolver.label")}</Label>
            <Input
              placeholder="letsencrypt"
              value={certResolver}
              onChange={(e) => setCertResolver(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-muted-foreground">
              {t("settings.certResolver.hint")}
            </p>
          </div>

          <hr className="border-border" />

          <div className="flex flex-col gap-1.5">
            <Label>{t("settings.appDomain.label")}</Label>
            <Input
              placeholder="app.example.com"
              value={appDomain}
              onChange={(e) => setAppDomain(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-muted-foreground">
              {t("settings.appDomain.hint")}
            </p>
          </div>

          <div className="flex items-center justify-between rounded-lg border p-3">
            <div className="flex flex-col gap-0.5">
              <Label>{t("settings.appTLS.label")}</Label>
              <p className="text-xs text-muted-foreground">
                {t("settings.appTLS.hint")}
              </p>
            </div>
            <Switch
              checked={appTLSEnabled}
              onCheckedChange={setAppTLSEnabled}
              disabled={isLoading || !appDomain}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>{t("settings.appBackendURL.label")}</Label>
            <Input
              placeholder="http://app:8080"
              value={appBackendURL}
              onChange={(e) => setAppBackendURL(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-muted-foreground">
              {t("settings.appBackendURL.hint")}
            </p>
          </div>

          <Button
            className="self-start"
            onClick={handleSave}
            disabled={isLoading || updateMutation.isPending}
          >
            {updateMutation.isPending ? t("settings.saving") : t("settings.save")}
          </Button>
        </div>
      </SectionCard>

      <SectionCard>
        <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
          <div className="max-w-2xl space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="outline">{t("settings.domainsCard.platformBadge")}</Badge>
              <Badge variant={appTLSEnabled ? "default" : "secondary"}>
                {appTLSEnabled ? "HTTPS" : "HTTP"}
              </Badge>
            </div>
            <div>
              <h3 className="text-base font-semibold">{t("settings.domainsCard.title")}</h3>
              <p className="text-sm text-muted-foreground">
                {t("settings.domainsCard.description")}
              </p>
            </div>
            <div className="rounded-2xl border border-border/70 bg-muted/30 p-4">
              <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                {t("settings.domainsCard.currentAppURL")}
              </p>
              <p className="mt-2 break-all font-mono text-sm text-foreground">
                {appDomain
                  ? `${appTLSEnabled ? "https" : "http"}://${appDomain}`
                  : t("settings.domainsCard.notConfigured")}
              </p>
            </div>
          </div>

          <Button asChild variant="outline" className="shrink-0">
            <Link to="/domains">
              <GlobeIcon className="mr-2 h-4 w-4" />
              {t("settings.domainsCard.manage")}
            </Link>
          </Button>
        </div>
      </SectionCard>
    </div>
  )
}
