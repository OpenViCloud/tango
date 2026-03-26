import type {
  GetRuntimeLogsRequestModel,
  RuntimeLogsModel,
} from "@/@types/models"
import type { ApiResponse } from "@/@types/models/common"

import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

const baseUrl = "/runtime/logs"

export const logService = {
  getRuntimeLogs: (params: GetRuntimeLogsRequestModel) =>
    api
      .get<ApiResponse<RuntimeLogsModel>>(baseUrl, { params })
      .then((res) => unwrapApiResponse(res.data)),
}
