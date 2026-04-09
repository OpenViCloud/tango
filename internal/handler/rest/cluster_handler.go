package rest

import (
	"context"
	"os"
	"time"

	"tango/internal/domain"
	response "tango/internal/handler/rest/response"
	appservices "tango/internal/application/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AnsibleRunner is the subset of ansible.Runner needed by ClusterHandler.
type AnsibleRunner interface {
	ProvisionCluster(
		clusterID string,
		servers []*domain.Server,
		nodes []domain.ClusterNode,
		k8sVersion, podCIDR string,
		clusterRepo domain.ClusterRepository,
		onKubeconfig func(ctx context.Context, kubeconfigPath string),
	)
	PurgeCluster(
		clusterID string,
		servers []*domain.Server,
		nodes []domain.ClusterNode,
		onDone func(err error),
	)
	PreviewInventory(servers []*domain.Server, nodes []domain.ClusterNode) (string, error)
}

type ClusterHandler struct {
	clusterRepo domain.ClusterRepository
	serverRepo  domain.ServerRepository
	runner      AnsibleRunner
	cipher      appservices.SecretCipher
}

func NewClusterHandler(
	clusterRepo domain.ClusterRepository,
	serverRepo domain.ServerRepository,
	runner AnsibleRunner,
	cipher appservices.SecretCipher,
) *ClusterHandler {
	return &ClusterHandler{
		clusterRepo: clusterRepo,
		serverRepo:  serverRepo,
		runner:      runner,
		cipher:      cipher,
	}
}

func (h *ClusterHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/clusters", h.List)
	rg.POST("/clusters", h.Bootstrap)
	rg.GET("/clusters/:id", h.Get)
	rg.POST("/clusters/:id/inventory-preview", h.InventoryPreview)
	rg.GET("/clusters/:id/kubeconfig", h.DownloadKubeconfig)
	rg.DELETE("/clusters/:id", h.Delete)
}

type clusterNodeRequest struct {
	ServerID string `json:"server_id" binding:"required"`
	Role     string `json:"role" binding:"required"` // master | worker
}

type bootstrapClusterRequest struct {
	Name       string               `json:"name" binding:"required"`
	Nodes      []clusterNodeRequest `json:"nodes" binding:"required,min=1"`
	K8sVersion string               `json:"k8s_version"`
	PodCIDR    string               `json:"pod_cidr"`
}

type inventoryPreviewRequest struct {
	Nodes []clusterNodeRequest `json:"nodes" binding:"required,min=1"`
}

type clusterNodeResponse struct {
	ServerID string `json:"server_id"`
	Role     string `json:"role"`
}

type clusterResponse struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Status     string                `json:"status"`
	ErrorMsg   string                `json:"error_msg,omitempty"`
	K8sVersion string                `json:"k8s_version"`
	PodCIDR    string                `json:"pod_cidr"`
	Nodes      []clusterNodeResponse `json:"nodes"`
	CreatedAt  time.Time             `json:"created_at"`
}

func toClusterResponse(c *domain.Cluster) clusterResponse {
	nodes := make([]clusterNodeResponse, 0, len(c.Nodes))
	for _, n := range c.Nodes {
		nodes = append(nodes, clusterNodeResponse{ServerID: n.ServerID, Role: string(n.Role)})
	}
	return clusterResponse{
		ID:         c.ID,
		Name:       c.Name,
		Status:     string(c.Status),
		ErrorMsg:   c.ErrorMsg,
		K8sVersion: c.K8sVersion,
		PodCIDR:    c.PodCIDR,
		Nodes:      nodes,
		CreatedAt:  c.CreatedAt,
	}
}

func (h *ClusterHandler) List(c *gin.Context) {
	clusters, err := h.clusterRepo.ListAll(c.Request.Context())
	if err != nil {
		_ = c.Error(response.Internal("list clusters failed"))
		return
	}
	resp := make([]clusterResponse, 0, len(clusters))
	for _, cl := range clusters {
		resp = append(resp, toClusterResponse(cl))
	}
	response.OK(c, resp)
}

func (h *ClusterHandler) Bootstrap(c *gin.Context) {
	var req bootstrapClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	// Validate: exactly one master
	masterCount := 0
	for _, n := range req.Nodes {
		if n.Role == string(domain.ClusterNodeRoleMaster) {
			masterCount++
		}
	}
	if masterCount != 1 {
		_ = c.Error(response.BadRequest("exactly one master node is required"))
		return
	}

	// Validate: all servers exist and are connected
	nodes := make([]domain.ClusterNode, 0, len(req.Nodes))
	allServers := make([]*domain.Server, 0, len(req.Nodes))
	for _, n := range req.Nodes {
		srv, err := h.serverRepo.GetByID(c.Request.Context(), n.ServerID)
		if err != nil {
			if err == domain.ErrServerNotFound {
				_ = c.Error(response.NotFound("server " + n.ServerID + " not found"))
				return
			}
			_ = c.Error(response.Internal("get server failed"))
			return
		}
		if srv.Status != domain.ServerStatusConnected {
			_ = c.Error(response.BadRequest("server " + srv.Name + " is not connected (run SSH ping first)"))
			return
		}
		allServers = append(allServers, srv)
		nodes = append(nodes, domain.ClusterNode{
			ServerID: n.ServerID,
			Role:     domain.ClusterNodeRole(n.Role),
		})
	}

	k8sVersion := req.K8sVersion
	if k8sVersion == "" {
		k8sVersion = "v1.30"
	}
	podCIDR := req.PodCIDR
	if podCIDR == "" {
		podCIDR = "192.168.0.0/16"
	}

	now := time.Now().UTC()
	cluster := &domain.Cluster{
		ID:         uuid.NewString(),
		Name:       req.Name,
		Status:     domain.ClusterStatusPending,
		Nodes:      nodes,
		K8sVersion: k8sVersion,
		PodCIDR:    podCIDR,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	saved, err := h.clusterRepo.Save(c.Request.Context(), cluster)
	if err != nil {
		_ = c.Error(response.Internal("save cluster failed"))
		return
	}

	// Mark provisioning immediately
	_ = h.clusterRepo.UpdateStatus(c.Request.Context(), saved.ID, domain.ClusterStatusProvisioning, "")

	// Launch provisioning async
	h.runner.ProvisionCluster(
		saved.ID,
		allServers,
		nodes,
		k8sVersion,
		podCIDR,
		h.clusterRepo,
		func(ctx context.Context, kubeconfigPath string) {
			data, err := os.ReadFile(kubeconfigPath)
			if err != nil {
				return
			}
			enc, err := h.cipher.Encrypt(ctx, string(data))
			if err != nil {
				return
			}
			_ = h.clusterRepo.UpdateKubeconfig(ctx, saved.ID, enc)
		},
	)

	response.Created(c, toClusterResponse(saved))
}

func (h *ClusterHandler) Get(c *gin.Context) {
	id := c.Param("id")
	cluster, err := h.clusterRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrClusterNotFound {
			_ = c.Error(response.NotFound("cluster not found"))
			return
		}
		_ = c.Error(response.Internal("get cluster failed"))
		return
	}
	response.OK(c, toClusterResponse(cluster))
}

func (h *ClusterHandler) InventoryPreview(c *gin.Context) {
	clusterID := c.Param("id")
	var req inventoryPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	_ = clusterID // used for context; future: validate cluster ownership

	nodes := make([]domain.ClusterNode, 0, len(req.Nodes))
	allServers := make([]*domain.Server, 0, len(req.Nodes))
	for _, n := range req.Nodes {
		srv, err := h.serverRepo.GetByID(c.Request.Context(), n.ServerID)
		if err != nil {
			_ = c.Error(response.NotFound("server " + n.ServerID + " not found"))
			return
		}
		allServers = append(allServers, srv)
		nodes = append(nodes, domain.ClusterNode{
			ServerID: n.ServerID,
			Role:     domain.ClusterNodeRole(n.Role),
		})
	}

	preview, err := h.runner.PreviewInventory(allServers, nodes)
	if err != nil {
		_ = c.Error(response.Internal("render inventory preview failed"))
		return
	}
	response.OK(c, gin.H{"inventory": preview})
}

func (h *ClusterHandler) DownloadKubeconfig(c *gin.Context) {
	id := c.Param("id")
	cluster, err := h.clusterRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrClusterNotFound {
			_ = c.Error(response.NotFound("cluster not found"))
			return
		}
		_ = c.Error(response.Internal("get cluster failed"))
		return
	}
	if cluster.Kubeconfig == "" {
		_ = c.Error(response.NotFound("kubeconfig not available yet"))
		return
	}
	plain, err := h.cipher.Decrypt(c.Request.Context(), cluster.Kubeconfig)
	if err != nil {
		_ = c.Error(response.Internal("decrypt kubeconfig failed"))
		return
	}
	c.Header("Content-Disposition", "attachment; filename=kubeconfig-"+cluster.Name+".yaml")
	c.Data(200, "application/x-yaml", []byte(plain))
}

func (h *ClusterHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	purge := c.Query("purge") == "true"

	if purge {
		cluster, err := h.clusterRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			if err == domain.ErrClusterNotFound {
				_ = c.Error(response.NotFound("cluster not found"))
				return
			}
			_ = c.Error(response.Internal("get cluster failed"))
			return
		}

		// Collect servers for all nodes (skip missing ones — best effort)
		servers := make([]*domain.Server, 0, len(cluster.Nodes))
		for _, n := range cluster.Nodes {
			srv, err := h.serverRepo.GetByID(c.Request.Context(), n.ServerID)
			if err != nil {
				continue
			}
			servers = append(servers, srv)
		}

		// Fire-and-forget: uninstall K8s on all nodes in background
		h.runner.PurgeCluster(id, servers, cluster.Nodes, nil)
	}

	if err := h.clusterRepo.Delete(c.Request.Context(), id); err != nil {
		if err == domain.ErrClusterNotFound {
			_ = c.Error(response.NotFound("cluster not found"))
			return
		}
		_ = c.Error(response.InternalCause(err, "delete cluster failed"))
		return
	}
	response.NoContent(c)
}

