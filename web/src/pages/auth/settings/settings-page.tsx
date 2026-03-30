import { useEffect, useState } from "react"
import { toast } from "sonner"
import { Settings2Icon, Trash2 } from "lucide-react"
import { useTranslation } from "react-i18next"

import { useGetSettings, useUpdateSettings } from "@/hooks/api/use-settings"
import {
  useGetBaseDomains,
  useCreateBaseDomain,
  useDeleteBaseDomain,
} from "@/hooks/api/use-base-domains"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"

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

  // Base domains state
  const { data: baseDomains = [] } = useGetBaseDomains()
  const createBaseDomainMutation = useCreateBaseDomain()
  const deleteBaseDomainMutation = useDeleteBaseDomain()
  const [newBaseDomain, setNewBaseDomain] = useState("")
  const [newBaseDomainWildcard, setNewBaseDomainWildcard] = useState(true)

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
      },
    )
  }

  const handleAddBaseDomain = () => {
    const domain = newBaseDomain.trim()
    if (!domain) return
    createBaseDomainMutation.mutate(
      { domain, wildcard_enabled: newBaseDomainWildcard },
      {
        onSuccess: () => {
          setNewBaseDomain("")
          setNewBaseDomainWildcard(true)
          toast.success(t("settings.baseDomains.added"))
        },
        onError: () => toast.error(t("settings.baseDomains.addFailed")),
      },
    )
  }

  const handleDeleteBaseDomain = (id: string) => {
    deleteBaseDomainMutation.mutate(id, {
      onSuccess: () => toast.success(t("settings.baseDomains.deleted")),
      onError: () => toast.error(t("settings.baseDomains.deleteFailed")),
    })
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<Settings2Icon className="size-6" />}
        title={t("settings.page.title")}
        description={t("settings.page.description")}
      />

      <SectionCard>
        <div className="flex flex-col gap-6 max-w-lg">
          {/* Public IP */}
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

          {/* Traefik network */}
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

          {/* Cert resolver */}
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

          {/* App Domain */}
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

          {/* App HTTPS */}
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

          {/* App Backend URL */}
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
            {updateMutation.isPending
              ? t("settings.saving")
              : t("settings.save")}
          </Button>
        </div>
      </SectionCard>

      {/* Base Domains */}
      <SectionCard>
        <div className="flex flex-col gap-4 max-w-lg">
          <div className="flex flex-col gap-0.5">
            <h3 className="text-sm font-semibold">{t("settings.baseDomains.title")}</h3>
            <p className="text-xs text-muted-foreground">{t("settings.baseDomains.description")}</p>
          </div>

          {baseDomains.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("settings.baseDomains.empty")}</p>
          ) : (
            <ul className="flex flex-col divide-y rounded-lg border">
              {baseDomains.map((bd) => (
                <li key={bd.id} className="flex items-center justify-between px-3 py-2">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-sm">{bd.domain}</span>
                    {bd.wildcard_enabled && (
                      <Badge variant="secondary" className="text-xs">
                        {t("settings.baseDomains.wildcardLabel")}
                      </Badge>
                    )}
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-destructive hover:text-destructive"
                    disabled={deleteBaseDomainMutation.isPending}
                    onClick={() => handleDeleteBaseDomain(bd.id)}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </li>
              ))}
            </ul>
          )}

          {/* Add form */}
          <div className="flex items-center gap-2">
            <Input
              placeholder={t("settings.baseDomains.addPlaceholder")}
              value={newBaseDomain}
              onChange={(e) => setNewBaseDomain(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleAddBaseDomain()}
              disabled={createBaseDomainMutation.isPending}
              className="max-w-xs"
            />
            <div className="flex items-center gap-1.5">
              <Switch
                id="new-bd-wildcard"
                checked={newBaseDomainWildcard}
                onCheckedChange={setNewBaseDomainWildcard}
                disabled={createBaseDomainMutation.isPending}
              />
              <Label htmlFor="new-bd-wildcard" className="text-sm cursor-pointer">
                {t("settings.baseDomains.wildcardLabel")}
              </Label>
            </div>
            <Button
              size="sm"
              onClick={handleAddBaseDomain}
              disabled={!newBaseDomain.trim() || createBaseDomainMutation.isPending}
            >
              Add
            </Button>
          </div>
        </div>
      </SectionCard>
    </div>
  )
}
