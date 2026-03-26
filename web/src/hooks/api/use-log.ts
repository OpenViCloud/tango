import { useQuery } from "@tanstack/react-query"

import type { GetRuntimeLogsRequestModel } from "@/@types/models"

import { logService } from "@/services/api/log-service"

export const LOG_QUERY_KEYS = {
  UseGetRuntimeLogs: (params: GetRuntimeLogsRequestModel) => [
    "UseGetRuntimeLogs",
    ...Object.values(params),
  ],
}

export const useGetRuntimeLogs = (params: GetRuntimeLogsRequestModel) =>
  useQuery({
    queryKey: LOG_QUERY_KEYS.UseGetRuntimeLogs(params),
    queryFn: () => logService.getRuntimeLogs(params),
  })
