import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import type { CreateBaseDomainModel } from "@/services/api/base-domain-service"
import { baseDomainService } from "@/services/api/base-domain-service"

export const BASE_DOMAINS_QUERY_KEY = ["base-domains"]

export const useGetBaseDomains = () =>
  useQuery({
    queryKey: BASE_DOMAINS_QUERY_KEY,
    queryFn: () => baseDomainService.list(),
  })

export const useCreateBaseDomain = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateBaseDomainModel) =>
      baseDomainService.create(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: BASE_DOMAINS_QUERY_KEY })
    },
  })
}

export const useDeleteBaseDomain = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => baseDomainService.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: BASE_DOMAINS_QUERY_KEY })
    },
  })
}

export const useCheckDomain = (domain: string) =>
  useQuery({
    queryKey: ["domain-check", domain],
    queryFn: () => baseDomainService.check(domain),
    enabled: !!domain && domain.length > 0,
    staleTime: 0,
  })
