import { api } from "@/lib/api"

export type SettingsModel = {
  public_ip: string
  base_domain: string
  wildcard_enabled: boolean
  traefik_network: string
  cert_resolver: string
  app_domain: string
  app_tls_enabled: boolean
  app_backend_url: string
}

export type UpdateSettingsModel = {
  public_ip?: string
  base_domain?: string
  wildcard_enabled?: boolean
  traefik_network?: string
  cert_resolver?: string
  app_domain?: string
  app_tls_enabled?: boolean
  app_backend_url?: string
}

export const settingsService = {
  getSettings: (): Promise<SettingsModel> =>
    api.get("/settings").then((r) => r.data),

  updateSettings: (payload: UpdateSettingsModel): Promise<SettingsModel> =>
    api.patch("/settings", payload).then((r) => r.data),
}
