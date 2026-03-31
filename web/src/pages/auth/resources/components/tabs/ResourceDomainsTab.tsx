import { useEffect, useState } from "react"
import { CheckCircle, Clock, Globe, Loader2, Plus, RefreshCw, Trash2, XCircle } from "lucide-react"
import { toast } from "sonner"
import { useTranslation } from "react-i18next"

import type { ResourceModel } from "@/@types/models"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import {
  useListResourceDomains,
  useAddResourceDomain,
  useRemoveResourceDomain,
  useVerifyResourceDomain,
} from "@/hooks/api/use-project"
import { useGetBaseDomains, useCheckDomain } from "@/hooks/api/use-base-domains"

const CUSTOM_DOMAIN_VALUE = "__custom__"

type Props = {
  resource: ResourceModel
}

export function ResourceDomainsTab({ resource }: Props) {
  const { t } = useTranslation()
  const availablePorts = resource.ports
    .filter((port) => port.internal_port > 0)
    .map((port) => ({
      value: String(port.internal_port),
      label: port.label?.trim()
        ? `${port.label} (${port.internal_port})`
        : t("domains.port.option", { port: port.internal_port }),
    }))

  // Base domains for the select combo
  const { data: baseDomains = [] } = useGetBaseDomains()

  // Whether to show combo (subdomain + base domain select) or plain text
  const hasBaseDomains = baseDomains.length > 0

  // "selected base domain" — undefined means custom mode when no base domains exist
  const [selectedBaseDomain, setSelectedBaseDomain] = useState<string>(
    hasBaseDomains ? baseDomains[0]?.id ?? CUSTOM_DOMAIN_VALUE : CUSTOM_DOMAIN_VALUE,
  )
  const isCustomMode = !hasBaseDomains || selectedBaseDomain === CUSTOM_DOMAIN_VALUE

  // Input values
  const [subdomainPrefix, setSubdomainPrefix] = useState("")
  const [customHost, setCustomHost] = useState("")
  const [tlsEnabled, setTlsEnabled] = useState(true)
  const [selectedTargetPort, setSelectedTargetPort] = useState("")

  // Debounced domain for availability check
  const [debouncedDomain, setDebouncedDomain] = useState("")

  const activeDomain = isCustomMode
    ? customHost.trim()
    : subdomainPrefix.trim()
      ? `${subdomainPrefix.trim()}.${baseDomains.find((bd) => bd.id === selectedBaseDomain)?.domain ?? ""}`
      : ""

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedDomain(activeDomain)
    }, 400)
    return () => clearTimeout(timer)
  }, [activeDomain])

  const { data: checkResult, isFetching: isChecking } = useCheckDomain(debouncedDomain)

  // Resource domain hooks
  const { data: domains = [], isLoading } = useListResourceDomains(resource.id)
  const addMutation = useAddResourceDomain(resource.id)
  const removeMutation = useRemoveResourceDomain(resource.id)
  const verifyMutation = useVerifyResourceDomain(resource.id)

  // Keep selectedBaseDomain in sync if base domains list changes
  useEffect(() => {
    if (hasBaseDomains && selectedBaseDomain === CUSTOM_DOMAIN_VALUE) {
      // keep custom if user explicitly chose it
    } else if (hasBaseDomains && !baseDomains.find((bd) => bd.id === selectedBaseDomain)) {
      setSelectedBaseDomain(baseDomains[0]?.id ?? CUSTOM_DOMAIN_VALUE)
    }
  }, [baseDomains, hasBaseDomains, selectedBaseDomain])

  useEffect(() => {
    if (availablePorts.length === 0) {
      setSelectedTargetPort("")
      return
    }
    if (!availablePorts.some((port) => port.value === selectedTargetPort)) {
      setSelectedTargetPort(availablePorts[0]?.value ?? "")
    }
  }, [availablePorts, selectedTargetPort])

  const handleAdd = () => {
    const host = activeDomain
    if (!host || !selectedTargetPort) return
    if (!isCustomMode && checkResult && !checkResult.available) return
    addMutation.mutate({ host, target_port: Number(selectedTargetPort), tls_enabled: tlsEnabled }, {
      onSuccess: () => {
        setSubdomainPrefix("")
        setCustomHost("")
        setDebouncedDomain("")
        toast.success(`Domain ${host} added`)
      },
      onError: () => toast.error("Failed to add domain"),
    })
  }

  const handleVerify = (domainId: string, host: string) => {
    verifyMutation.mutate(domainId, {
      onSuccess: (res) => {
        if (res.verified) {
          toast.success(`${host} verified`)
        } else {
          toast.error(`DNS not pointing to server yet. Resolved: ${res.resolved_ips?.join(", ") || "none"}`)
        }
      },
    })
  }

  const handleRemove = (domainId: string, host: string) => {
    removeMutation.mutate(domainId, {
      onSuccess: () => toast.success(`${host} removed`),
    })
  }

  const canAdd = (() => {
    if (!activeDomain || !selectedTargetPort || addMutation.isPending) return false
    if (!isCustomMode) {
      // For base-domain mode, require availability check to pass
      if (isChecking || debouncedDomain !== activeDomain) return false
      if (!checkResult || !checkResult.available) return false
    }
    return true
  })()

  const renderAvailabilityIndicator = () => {
    if (!activeDomain) return null
    if (isChecking || debouncedDomain !== activeDomain) {
      return (
        <span className="flex items-center gap-1 text-xs text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          {t("domains.checking")}
        </span>
      )
    }
    if (!checkResult) return null
    if (checkResult.available) {
      return (
        <span className="flex items-center gap-1 text-xs text-green-600">
          <CheckCircle className="h-3.5 w-3.5" />
          {t("domains.available")}
        </span>
      )
    }
    return (
      <span className="flex items-center gap-1 text-xs text-destructive">
        <XCircle className="h-3.5 w-3.5" />
        {t("domains.inUse")}
        {checkResult.used_by_resource_name ? ` (${checkResult.used_by_resource_name})` : ""}
      </span>
    )
  }

  return (
    <main className="min-w-0 flex-1 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-col gap-4">

        {/* Add domain */}
        <div className="rounded-2xl border bg-card p-5">
          <h2 className="mb-1 text-base font-semibold">Add Custom Domain</h2>
          <p className="mb-4 text-sm text-muted-foreground">
            Point your domain's DNS A record to the server IP, then add it here.
          </p>

            <div className="flex flex-col gap-2">
            <div className="flex flex-wrap items-center gap-3 rounded-xl border border-border/60 bg-muted/30 px-3 py-2">
              <div className="flex items-center gap-2">
                <Label htmlFor="resource-domain-tls" className="text-sm">
                  TLS
                </Label>
                <Switch
                  id="resource-domain-tls"
                  checked={tlsEnabled}
                  onCheckedChange={setTlsEnabled}
                  disabled={addMutation.isPending}
                />
              </div>
              <Badge variant="outline" className="font-mono uppercase">
                {tlsEnabled ? "https" : "http"}
              </Badge>
              <span className="text-xs text-muted-foreground">
                {tlsEnabled ? "Traefik will route over HTTPS." : "Traefik will route over HTTP only."}
              </span>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              {hasBaseDomains ? (
                <>
                  {/* Base domain select */}
                  <Select value={selectedBaseDomain} onValueChange={setSelectedBaseDomain}>
                    <SelectTrigger className="w-48">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {baseDomains.map((bd) => (
                        <SelectItem key={bd.id} value={bd.id}>
                          {bd.domain}
                        </SelectItem>
                      ))}
                      <SelectItem value={CUSTOM_DOMAIN_VALUE}>
                        {t("domains.customDomain")}
                      </SelectItem>
                    </SelectContent>
                  </Select>

                  {isCustomMode ? (
                    /* Custom domain input */
                    <Input
                      placeholder="api.example.com"
                      value={customHost}
                      onChange={(e) => setCustomHost(e.target.value)}
                      onKeyDown={(e) => e.key === "Enter" && handleAdd()}
                      disabled={addMutation.isPending}
                      className="max-w-sm"
                    />
                  ) : (
                    /* Subdomain prefix input */
                    <div className="flex items-center gap-1">
                      <Input
                        placeholder={t("domains.subdomainPlaceholder")}
                        value={subdomainPrefix}
                        onChange={(e) => setSubdomainPrefix(e.target.value)}
                        onKeyDown={(e) => e.key === "Enter" && handleAdd()}
                        disabled={addMutation.isPending}
                        className="w-36"
                      />
                      <span className="text-sm text-muted-foreground">
                        .{baseDomains.find((bd) => bd.id === selectedBaseDomain)?.domain}
                      </span>
                    </div>
                  )}
                </>
              ) : (
                /* No base domains — plain text input */
                <Input
                  placeholder="api.example.com"
                  value={customHost}
                  onChange={(e) => setCustomHost(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleAdd()}
                  disabled={addMutation.isPending}
                  className="max-w-sm"
                />
              )}

              <Button
                onClick={handleAdd}
                disabled={!canAdd}
                size="sm"
              >
                <Plus className="mr-1.5 h-3.5 w-3.5" />
                Add
              </Button>
            </div>

            {/* Availability indicator (only for base domain combo mode) */}
            {!isCustomMode && renderAvailabilityIndicator()}

            {availablePorts.length === 0 ? (
              <p className="text-xs text-destructive">
                {t("domains.port.empty")}
              </p>
            ) : availablePorts.length === 1 ? (
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span>{t("domains.port.targetLabel")}</span>
                <Badge variant="outline" className="font-mono">
                  {availablePorts[0]?.label}
                </Badge>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <Label className="text-xs text-muted-foreground">
                  {t("domains.port.targetLabel")}
                </Label>
                <Select value={selectedTargetPort} onValueChange={setSelectedTargetPort}>
                  <SelectTrigger className="w-[220px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {availablePorts.map((port) => (
                      <SelectItem key={port.value} value={port.value}>
                        {port.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>
        </div>

        {/* Domain list */}
        <div className="rounded-2xl border bg-card">
          <div className="border-b px-5 py-3">
            <h2 className="text-base font-semibold">Domains</h2>
          </div>

          {isLoading ? (
            <div className="px-5 py-8 text-sm text-muted-foreground">Loading…</div>
          ) : domains.length === 0 ? (
            <div className="px-5 py-8 text-sm text-muted-foreground">No domains configured.</div>
          ) : (
            <ul className="divide-y">
              {domains.map((d) => (
                <li key={d.id} className="flex items-center gap-3 px-5 py-3">
                  <Globe className="h-4 w-4 shrink-0 text-muted-foreground" />

                  <span className="min-w-0 flex-1 truncate font-mono text-sm">{d.host}</span>

                  <div className="flex shrink-0 items-center gap-2">
                    <Badge variant="outline" className="text-xs font-mono">
                      {t("domains.port.badge", { port: d.target_port })}
                    </Badge>
                    <Badge variant="outline" className="text-xs font-mono uppercase">
                      {d.tls_enabled ? "https" : "http"}
                    </Badge>
                    {d.type === "auto" ? (
                      <Badge variant="secondary" className="text-xs">auto</Badge>
                    ) : d.verified ? (
                      <span className="flex items-center gap-1 text-xs text-green-600">
                        <CheckCircle className="h-3.5 w-3.5" />
                        verified
                      </span>
                    ) : (
                      <span className="flex items-center gap-1 text-xs text-yellow-600">
                        <Clock className="h-3.5 w-3.5" />
                        unverified
                      </span>
                    )}

                    {d.type === "custom" && !d.verified && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7"
                        disabled={verifyMutation.isPending}
                        onClick={() => handleVerify(d.id, d.host)}
                        title="Check DNS"
                      >
                        <RefreshCw className="h-3.5 w-3.5" />
                      </Button>
                    )}

                    {d.type === "custom" && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 text-destructive hover:text-destructive"
                        disabled={removeMutation.isPending}
                        onClick={() => handleRemove(d.id, d.host)}
                        title="Remove"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>

        {resource.status === "running" && domains.length > 0 && (
          <p className="text-xs text-muted-foreground">
            Verified domains are applied instantly. Start the resource to activate unverified domains.
          </p>
        )}
      </div>
    </main>
  )
}
