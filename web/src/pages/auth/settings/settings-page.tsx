import { useEffect, useState } from "react"
import { toast } from "sonner"
import { Settings2Icon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { useGetSettings, useUpdateSettings } from "@/hooks/api/use-settings"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"

export function SettingsPage() {
  const { t } = useTranslation()
  const { data: settings, isLoading } = useGetSettings()
  const updateMutation = useUpdateSettings()

  const [publicIp, setPublicIp] = useState("")
  const [baseDomain, setBaseDomain] = useState("")
  const [wildcardEnabled, setWildcardEnabled] = useState(true)
  const [traefikNetwork, setTraefikNetwork] = useState("")
  const [certResolver, setCertResolver] = useState("letsencrypt")

  useEffect(() => {
    if (settings) {
      setPublicIp(settings.public_ip ?? "")
      setBaseDomain(settings.base_domain ?? "")
      setWildcardEnabled(settings.wildcard_enabled ?? true)
      setTraefikNetwork(settings.traefik_network ?? "")
      setCertResolver(settings.cert_resolver ?? "letsencrypt")
    }
  }, [settings])

  const handleSave = () => {
    updateMutation.mutate(
      {
        public_ip: publicIp,
        base_domain: baseDomain,
        wildcard_enabled: wildcardEnabled,
        traefik_network: traefikNetwork,
        cert_resolver: certResolver,
      },
      {
        onSuccess: () => toast.success(t("settings.saved")),
        onError: () => toast.error(t("settings.saveFailed")),
      },
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

          {/* Base domain */}
          <div className="flex flex-col gap-1.5">
            <Label>{t("settings.baseDomain.label")}</Label>
            <Input
              placeholder="example.com"
              value={baseDomain}
              onChange={(e) => setBaseDomain(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-muted-foreground">
              {t("settings.baseDomain.hint")}
            </p>
          </div>

          {/* Wildcard enabled */}
          <div className="flex items-center justify-between rounded-lg border p-3">
            <div className="flex flex-col gap-0.5">
              <Label>{t("settings.wildcard.label")}</Label>
              <p className="text-xs text-muted-foreground">
                {t("settings.wildcard.hint")}
              </p>
            </div>
            <Switch
              checked={wildcardEnabled}
              onCheckedChange={setWildcardEnabled}
              disabled={isLoading}
            />
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
    </div>
  )
}
