import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import type {
  CreateCloudflareConnectionModel,
  UpdateCloudflareConnectionModel,
} from "@/@types/models"
import { cloudflareService } from "@/services/api/cloudflare-service"

export const CLOUDFLARE_QUERY_KEYS = {
  list: () => ["cloudflare", "connections"],
  detail: (id: string) => ["cloudflare", "connections", id],
}

export const useGetCloudflareConnections = () =>
  useQuery({
    queryKey: CLOUDFLARE_QUERY_KEYS.list(),
    queryFn: () => cloudflareService.list(),
  })

export const useGetCloudflareConnection = (id: string) =>
  useQuery({
    queryKey: CLOUDFLARE_QUERY_KEYS.detail(id),
    queryFn: () => cloudflareService.getById(id),
    enabled: Boolean(id),
  })

export const useCreateCloudflareConnection = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateCloudflareConnectionModel) =>
      cloudflareService.create(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CLOUDFLARE_QUERY_KEYS.list() })
    },
  })
}

export const useUpdateCloudflareConnection = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      payload,
    }: {
      id: string
      payload: UpdateCloudflareConnectionModel
    }) => cloudflareService.update(id, payload),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: CLOUDFLARE_QUERY_KEYS.list() })
      queryClient.invalidateQueries({
        queryKey: CLOUDFLARE_QUERY_KEYS.detail(variables.id),
      })
    },
  })
}
