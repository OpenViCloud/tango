import { useState } from "react"
import { toast } from "sonner"
import { CheckCircle2Icon, GlobeIcon, LockIcon, NetworkIcon, RotateCwIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { useGetSettings, useRestartTraefik, useUpdateSettings } from "@/hooks/api/use-settings"
import { PageHeaderCard } from "@/components/share/cards/page-header-card"
import { SectionCard } from "@/components/share/cards/section-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { appIcons } from "@/lib/icons"
import type { SettingsModel } from "@/services/api/settings-service"

const SettingsIcon = appIcons.settings

function createSettingsDraft(settings?: SettingsModel) {
  return {
    publicIp: settings?.public_ip ?? "",
    traefikNetwork: settings?.traefik_network ?? "tango_net",
    certResolver: settings?.cert_resolver ?? "letsencrypt",
    appDomain: settings?.app_domain ?? "",
    appTLSEnabled: settings?.app_tls_enabled ?? false,
    appBackendURL: settings?.app_backend_url ?? "http://app:8080",
    acmeEmail: settings?.acme_email ?? "",
  }
}

export function SettingsPage() {
  const { t } = useTranslation()
  const { data: settings, isLoading } = useGetSettings()

  return (
    <div className="flex flex-col gap-6">
      <PageHeaderCard
        icon={<SettingsIcon />}
        title={t("settings.page.title")}
        description={t("settings.page.description")}
      />

      <SettingsForm
        key={JSON.stringify(settings ?? {})}
        settings={settings}
        isLoading={isLoading}
      />
    </div>
  )
}

function SettingsForm({
  settings,
  isLoading,
}: {
  settings?: SettingsModel
  isLoading: boolean
}) {
  const { t } = useTranslation()
  const updateMutation = useUpdateSettings()
  const restartMutation = useRestartTraefik()
  const [draft, setDraft] = useState(() => createSettingsDraft(settings))

  const handleSave = () => {
    updateMutation.mutate(
      {
        public_ip: draft.publicIp,
        traefik_network: draft.traefikNetwork,
        cert_resolver: draft.certResolver,
        app_domain: draft.appDomain,
        app_tls_enabled: draft.appTLSEnabled,
        app_backend_url: draft.appBackendURL,
        acme_email: draft.acmeEmail,
      },
      {
        onSuccess: () => toast.success(t("settings.saved")),
        onError: () => toast.error(t("settings.saveFailed")),
      }
    )
  }

  const appUrlPreview = draft.appDomain
    ? `${draft.appTLSEnabled ? "https" : "http"}://${draft.appDomain}`
    : "Not configured"
  const checks = [
    { label: t("settings.publicIp.label"), ready: Boolean(draft.publicIp.trim()) },
    { label: t("settings.appDomain.label"), ready: Boolean(draft.appDomain.trim()) },
    {
      label: t("settings.traefikNetwork.label"),
      ready: Boolean(draft.traefikNetwork.trim()),
    },
    {
      label: t("settings.acmeEmail.label"),
      ready: Boolean(draft.acmeEmail.trim()),
      hidden: !draft.appTLSEnabled,
    },
  ]

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_360px]">
      <div className="flex flex-col gap-6">
        <SectionCard
          icon={<NetworkIcon className="size-5" />}
          title="Platform"
          description="Core networking values used by Traefik and platform-wide routing."
          contentClassName="pt-1"
        >
          <FieldGroup>
            <Field>
              <FieldLabel>{t("settings.publicIp.label")}</FieldLabel>
              <Input
                className="font-mono"
                placeholder="115.74.113.215"
                value={draft.publicIp}
                onChange={(e) =>
                  setDraft((current) => ({ ...current, publicIp: e.target.value }))
                }
                disabled={isLoading}
              />
              <FieldDescription>{t("settings.publicIp.hint")}</FieldDescription>
            </Field>

            <Field>
              <FieldLabel>{t("settings.traefikNetwork.label")}</FieldLabel>
              <Input
                placeholder="tango_net"
                value={draft.traefikNetwork}
                onChange={(e) =>
                  setDraft((current) => ({
                    ...current,
                    traefikNetwork: e.target.value,
                  }))
                }
                disabled={isLoading}
              />
              <FieldDescription>{t("settings.traefikNetwork.hint")}</FieldDescription>
            </Field>

            <Field>
              <FieldLabel>{t("settings.certResolver.label")}</FieldLabel>
              <Input
                placeholder="letsencrypt"
                value={draft.certResolver}
                onChange={(e) =>
                  setDraft((current) => ({
                    ...current,
                    certResolver: e.target.value,
                  }))
                }
                disabled={isLoading}
              />
              <FieldDescription>{t("settings.certResolver.hint")}</FieldDescription>
            </Field>
          </FieldGroup>
        </SectionCard>

        <SectionCard
          icon={<LockIcon className="size-5" />}
          title="HTTPS / Let's Encrypt"
          description="ACME email used by Traefik to obtain TLS certificates."
          contentClassName="pt-1"
        >
          <FieldGroup>
            <Field>
              <FieldLabel>{t("settings.acmeEmail.label")}</FieldLabel>
              <Input
                type="email"
                className="font-mono"
                placeholder="admin@example.com"
                value={draft.acmeEmail}
                onChange={(e) =>
                  setDraft((current) => ({ ...current, acmeEmail: e.target.value }))
                }
                disabled={isLoading}
              />
              <FieldDescription>{t("settings.acmeEmail.hint")}</FieldDescription>
            </Field>
          </FieldGroup>

          <div className="mt-4 flex items-center justify-between rounded-xl border border-dashed border-border/70 px-4 py-3">
            <div>
              <p className="text-sm font-medium">{t("settings.restartTraefik.label")}</p>
              <p className="text-xs text-muted-foreground">{t("settings.restartTraefik.hint")}</p>
            </div>
            <Button
              variant="outline"
              size="sm"
              disabled={restartMutation.isPending}
              onClick={() =>
                restartMutation.mutate(undefined, {
                  onSuccess: () => toast.success(t("settings.restartTraefik.success")),
                  onError: () => toast.error(t("settings.restartTraefik.error")),
                })
              }
            >
              <RotateCwIcon className={`size-4 ${restartMutation.isPending ? "animate-spin" : ""}`} />
              {restartMutation.isPending
                ? t("settings.restartTraefik.pending")
                : t("settings.restartTraefik.label")}
            </Button>
          </div>
        </SectionCard>

        <SectionCard
          icon={<GlobeIcon className="size-5" />}
          title="App ingress"
          description="Configure the public entrypoint and internal backend target for the Tango app."
          contentClassName="pt-1"
        >
          <FieldGroup>
            <Field>
              <FieldLabel>{t("settings.appDomain.label")}</FieldLabel>
              <Input
                placeholder="app.example.com"
                value={draft.appDomain}
                onChange={(e) =>
                  setDraft((current) => ({ ...current, appDomain: e.target.value }))
                }
                disabled={isLoading}
              />
              <FieldDescription>{t("settings.appDomain.hint")}</FieldDescription>
            </Field>

            <FieldSeparator />

            <Field orientation="horizontal" className="rounded-xl border px-4 py-3">
              <FieldLabel className="gap-3 border-none p-0">
                <FieldContent>
                  <div className="text-sm font-medium">{t("settings.appTLS.label")}</div>
                  <FieldDescription className="mt-0 text-xs">
                    {t("settings.appTLS.hint")}
                  </FieldDescription>
                </FieldContent>
              </FieldLabel>
              <div className="flex items-center gap-3">
                <Badge variant={draft.appTLSEnabled ? "default" : "secondary"}>
                  {draft.appTLSEnabled ? "HTTPS" : "HTTP"}
                </Badge>
                <Switch
                  checked={draft.appTLSEnabled}
                  onCheckedChange={(checked) =>
                    setDraft((current) => ({ ...current, appTLSEnabled: checked }))
                  }
                  disabled={isLoading || !draft.appDomain || !draft.acmeEmail}
                />
              </div>
            </Field>

            <Field>
              <FieldLabel>{t("settings.appBackendURL.label")}</FieldLabel>
              <Input
                className="font-mono"
                placeholder="http://app:8080"
                value={draft.appBackendURL}
                onChange={(e) =>
                  setDraft((current) => ({
                    ...current,
                    appBackendURL: e.target.value,
                  }))
                }
                disabled={isLoading}
              />
              <FieldDescription>{t("settings.appBackendURL.hint")}</FieldDescription>
            </Field>
          </FieldGroup>
        </SectionCard>

        <div className="flex items-center justify-end">
          <Button
            className="min-w-32"
            onClick={handleSave}
            disabled={isLoading || updateMutation.isPending}
          >
            {updateMutation.isPending ? t("settings.saving") : t("settings.save")}
          </Button>
        </div>
      </div>

      <Card className="h-fit border-none bg-card shadow-panel">
        <CardHeader className="gap-4">
          <div className="flex items-center justify-between gap-3">
            <div>
              <CardTitle>Current config</CardTitle>
              <CardDescription>
                Live preview of the current routing setup before saving.
              </CardDescription>
            </div>
            <Badge variant={draft.appTLSEnabled ? "default" : "secondary"}>
              {draft.appTLSEnabled ? "HTTPS" : "HTTP"}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-5">
          <div className="rounded-2xl border border-border/70 bg-muted/20 p-4">
            <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
              App URL
            </p>
            <p className="mt-2 break-all font-mono text-sm text-foreground">
              {appUrlPreview}
            </p>
          </div>

          <div className="rounded-2xl border border-border/70 bg-muted/20 p-4">
            <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
              Backend target
            </p>
            <p className="mt-2 break-all font-mono text-sm text-foreground">
              {draft.appBackendURL || "Not configured"}
            </p>
          </div>

          <div className="flex flex-col gap-3 rounded-2xl border border-border/70 bg-background/60 p-4">
            {checks.filter((item) => !item.hidden).map((item) => (
              <div
                key={item.label}
                className="flex items-center justify-between gap-3"
              >
                <div className="flex items-center gap-2 text-sm">
                  <CheckCircle2Icon
                    className={`size-4 ${item.ready ? "text-primary" : "text-muted-foreground"}`}
                  />
                  <span>{item.label}</span>
                </div>
                <Badge variant={item.ready ? "default" : "secondary"}>
                  {item.ready ? "Ready" : "Missing"}
                </Badge>
              </div>
            ))}
          </div>

          <div className="rounded-2xl border border-dashed border-border/70 p-4">
            <p className="text-sm font-medium">Routing notes</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Enable TLS only after the domain resolves correctly and the cert
              resolver is available in Traefik.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
