import type {
  BuildJobModel,
  CreateBuildJobModel,
  GetBuildJobListRequestModel,
} from "@/@types/models"
import type { ApiResponse, PagedResponse } from "@/@types/models/common"

import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

const baseUrl = "/builds"

export const buildService = {
  create: (payload: CreateBuildJobModel) =>
    api
      .post<ApiResponse<BuildJobModel>>(baseUrl, payload)
      .then((res) => unwrapApiResponse(res.data)),

  upload: (file: File, imageTag: string, buildMode: "auto" | "dockerfile") => {
    const form = new FormData()
    form.append("archive", file)
    form.append("image_tag", imageTag)
    form.append("build_mode", buildMode)
    return api
      .post<ApiResponse<BuildJobModel>>(`${baseUrl}/upload`, form, {
        headers: { "Content-Type": "multipart/form-data" },
      })
      .then((res) => unwrapApiResponse(res.data))
  },

  getById: (id: string) =>
    api
      .get<ApiResponse<BuildJobModel>>(`${baseUrl}/${id}`)
      .then((res) => unwrapApiResponse(res.data)),

  getList: (params: GetBuildJobListRequestModel) =>
    api
      .get<ApiResponse<PagedResponse<BuildJobModel>>>(baseUrl, { params })
      .then((res) => unwrapApiResponse(res.data)),

  cancel: (id: string) =>
    api
      .post<ApiResponse<BuildJobModel>>(`${baseUrl}/${id}/cancel`)
      .then((res) => unwrapApiResponse(res.data)),
}
