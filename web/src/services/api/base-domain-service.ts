import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

export type BaseDomainModel = {
  id: string
  domain: string
  wildcard_enabled: boolean
  created_at: string
}

export type CreateBaseDomainModel = {
  domain: string
  wildcard_enabled: boolean
}

export type DomainCheckResult = {
  available: boolean
  used_by_resource_id?: string
  used_by_resource_name?: string
}

export const baseDomainService = {
  list: (): Promise<BaseDomainModel[]> =>
    api.get("/settings/base-domains").then((r) => unwrapApiResponse(r.data)),

  create: (payload: CreateBaseDomainModel): Promise<BaseDomainModel> =>
    api.post("/settings/base-domains", payload).then((r) => unwrapApiResponse(r.data)),

  delete: (id: string): Promise<void> =>
    api.delete(`/settings/base-domains/${id}`).then((r) => r.data),

  check: (domain: string): Promise<DomainCheckResult> =>
    api.get(`/domains/check?domain=${encodeURIComponent(domain)}`).then((r) => r.data),
}
