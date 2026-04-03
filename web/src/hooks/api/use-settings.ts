import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import type { UpdateSettingsModel } from "@/services/api/settings-service"
import { settingsService } from "@/services/api/settings-service"

export const SETTINGS_QUERY_KEY = ["settings"]

export const useGetSettings = () =>
  useQuery({
    queryKey: SETTINGS_QUERY_KEY,
    queryFn: () => settingsService.getSettings(),
  })

export const useUpdateSettings = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: UpdateSettingsModel) =>
      settingsService.updateSettings(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: SETTINGS_QUERY_KEY })
    },
  })
}

export const useRestartTraefik = () =>
  useMutation({
    mutationFn: () => settingsService.restartTraefik(),
  })
