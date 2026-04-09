import type { ApiResponse } from "@/@types/models/common"
import type {
  BootstrapClusterModel,
  ClusterModel,
} from "@/@types/models/server"
import { api } from "@/lib/api"
import { unwrapApiResponse } from "@/lib/api-response"

export const clusterService = {
  list: (): Promise<ClusterModel[]> =>
    api
      .get<ApiResponse<ClusterModel[]>>("/clusters")
      .then((res) => unwrapApiResponse(res.data)),

  getById: (id: string): Promise<ClusterModel> =>
    api
      .get<ApiResponse<ClusterModel>>(`/clusters/${id}`)
      .then((res) => unwrapApiResponse(res.data)),

  bootstrap: (payload: BootstrapClusterModel): Promise<ClusterModel> =>
    api
      .post<ApiResponse<ClusterModel>>("/clusters", payload)
      .then((res) => unwrapApiResponse(res.data)),

  inventoryPreview: (
    clusterId: string,
    nodes: BootstrapClusterModel["nodes"],
  ): Promise<{ inventory: string }> =>
    api
      .post<ApiResponse<{ inventory: string }>>(
        `/clusters/${clusterId}/inventory-preview`,
        { nodes },
      )
      .then((res) => unwrapApiResponse(res.data)),

  downloadKubeconfig: (id: string): Promise<Blob> =>
    api
      .get(`/clusters/${id}/kubeconfig`, { responseType: "blob" })
      .then((res) => res.data as Blob),

  delete: (id: string, purge?: boolean): Promise<void> =>
    api
      .delete(`/clusters/${id}`, { params: purge ? { purge: "true" } : undefined })
      .then(() => undefined),
}
