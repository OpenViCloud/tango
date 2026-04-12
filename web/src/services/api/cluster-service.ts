import type { ApiResponse } from "@/@types/models/common"
import type {
  BootstrapClusterModel,
  ClusterModel,
  ClusterTunnelModel,
  ImportClusterTunnelModel,
  KubeNamespace,
  KubePod,
  KubeService,
  KubePersistentVolume,
  KubePersistentVolumeClaim,
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

  importTunnel: (
    id: string,
    payload: ImportClusterTunnelModel,
  ): Promise<ClusterTunnelModel> =>
    api
      .post<ApiResponse<ClusterTunnelModel>>(`/clusters/${id}/tunnels/import`, payload)
      .then((res) => unwrapApiResponse(res.data)),

  // ── Kubernetes resource endpoints ──────────────────────────────────────────

  listNamespaces: (id: string): Promise<KubeNamespace[]> =>
    api
      .get<ApiResponse<KubeNamespace[]>>(`/clusters/${id}/namespaces`)
      .then((res) => unwrapApiResponse(res.data)),

  listPods: (id: string, namespace?: string): Promise<KubePod[]> =>
    api
      .get<ApiResponse<KubePod[]>>(`/clusters/${id}/pods`, {
        params: namespace ? { namespace } : undefined,
      })
      .then((res) => unwrapApiResponse(res.data)),

  listServices: (id: string, namespace?: string): Promise<KubeService[]> =>
    api
      .get<ApiResponse<KubeService[]>>(`/clusters/${id}/services`, {
        params: namespace ? { namespace } : undefined,
      })
      .then((res) => unwrapApiResponse(res.data)),

  listPersistentVolumes: (id: string): Promise<KubePersistentVolume[]> =>
    api
      .get<ApiResponse<KubePersistentVolume[]>>(`/clusters/${id}/volumes`)
      .then((res) => unwrapApiResponse(res.data)),

  listPersistentVolumeClaims: (
    id: string,
    namespace?: string,
  ): Promise<KubePersistentVolumeClaim[]> =>
    api
      .get<ApiResponse<KubePersistentVolumeClaim[]>>(
        `/clusters/${id}/volume-claims`,
        { params: namespace ? { namespace } : undefined },
      )
      .then((res) => unwrapApiResponse(res.data)),

  createPod: (
    id: string,
    namespace: string,
    payload: { name: string; image: string; labels?: Record<string, string> },
  ): Promise<KubePod> =>
    api
      .post<ApiResponse<KubePod>>(`/clusters/${id}/pods`, payload, {
        params: { namespace },
      })
      .then((res) => unwrapApiResponse(res.data)),

  deletePod: (id: string, namespace: string, name: string): Promise<void> =>
    api
      .delete(`/clusters/${id}/pods/${name}`, { params: { namespace } })
      .then(() => undefined),

  deleteService: (id: string, namespace: string, name: string): Promise<void> =>
    api
      .delete(`/clusters/${id}/services/${name}`, { params: { namespace } })
      .then(() => undefined),

  createService: (
    id: string,
    namespace: string,
    payload: {
      name: string
      type: string
      selector: Record<string, string>
      ports: Array<{ name: string; port: number; target_port: string; protocol: string }>
    },
  ): Promise<KubeService> =>
    api
      .post<ApiResponse<KubeService>>(`/clusters/${id}/services`, payload, {
        params: { namespace },
      })
      .then((res) => unwrapApiResponse(res.data)),
}
