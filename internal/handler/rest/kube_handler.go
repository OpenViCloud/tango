package rest

import (
	"net/http"

	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

// KubeHandler exposes Kubernetes resource management endpoints per cluster.
// Routes are nested under /clusters/:id so the cluster is always the context.
type KubeHandler struct {
	factory     domain.KubeClientFactory
	clusterRepo domain.ClusterRepository
}

// NewKubeHandler creates a new KubeHandler.
func NewKubeHandler(factory domain.KubeClientFactory, clusterRepo domain.ClusterRepository) *KubeHandler {
	return &KubeHandler{factory: factory, clusterRepo: clusterRepo}
}

// Register mounts all Kubernetes resource routes onto the given router group.
func (h *KubeHandler) Register(rg *gin.RouterGroup) {
	c := rg.Group("/clusters/:id")
	c.GET("/namespaces", h.ListNamespaces)
	c.GET("/pods", h.ListPods)
	c.POST("/pods", h.CreatePod)
	c.DELETE("/pods/:name", h.DeletePod)
	c.GET("/services", h.ListServices)
	c.POST("/services", h.CreateService)
	c.DELETE("/services/:name", h.DeleteService)
	c.GET("/volumes", h.ListPersistentVolumes)
	c.GET("/volume-claims", h.ListPersistentVolumeClaims)
}

// ── request / response types ─────────────────────────────────────────────────

type createPodRequest struct {
	Name   string            `json:"name"   binding:"required"`
	Image  string            `json:"image"  binding:"required"`
	Labels map[string]string `json:"labels"`
}

type createServiceRequest struct {
	Name      string                   `json:"name"      binding:"required"`
	Type      string                   `json:"type"      binding:"required"` // ClusterIP | NodePort | LoadBalancer
	Selector  map[string]string        `json:"selector"  binding:"required"`
	Ports     []domain.KubeServicePort `json:"ports"     binding:"required,min=1"`
}

// ── helpers ───────────────────────────────────────────────────────────────────

// getReadyClient validates the cluster and returns its KubeClient.
// Returns false (and writes the error to c) when the cluster is unavailable.
func (h *KubeHandler) getReadyClient(c *gin.Context) (domain.KubeClient, bool) {
	clusterID := c.Param("id")

	cluster, err := h.clusterRepo.GetByID(c.Request.Context(), clusterID)
	if err != nil {
		if err == domain.ErrClusterNotFound {
			_ = c.Error(response.NotFound("cluster not found"))
			return nil, false
		}
		_ = c.Error(response.Internal("get cluster failed"))
		return nil, false
	}
	if cluster.Status != domain.ClusterStatusReady {
		_ = c.Error(&gin.Error{
			Err:  nil,
			Type: gin.ErrorTypePublic,
			Meta: gin.H{
				"success": false,
				"error":   "cluster is not ready (status: " + string(cluster.Status) + ")",
			},
		})
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "cluster is not ready (status: " + string(cluster.Status) + ")",
		})
		return nil, false
	}

	client, err := h.factory.GetClient(c.Request.Context(), clusterID)
	if err != nil {
		_ = c.Error(response.Internal("build kube client failed: " + err.Error()))
		return nil, false
	}
	return client, true
}

// ── handlers ──────────────────────────────────────────────────────────────────

func (h *KubeHandler) ListNamespaces(c *gin.Context) {
	client, ok := h.getReadyClient(c)
	if !ok {
		return
	}
	namespaces, err := client.ListNamespaces(c.Request.Context())
	if err != nil {
		_ = c.Error(response.Internal("list namespaces failed: " + err.Error()))
		return
	}
	response.OK(c, namespaces)
}

func (h *KubeHandler) ListPods(c *gin.Context) {
	client, ok := h.getReadyClient(c)
	if !ok {
		return
	}
	namespace := c.DefaultQuery("namespace", "")
	pods, err := client.ListPods(c.Request.Context(), namespace)
	if err != nil {
		_ = c.Error(response.Internal("list pods failed: " + err.Error()))
		return
	}
	response.OK(c, pods)
}

func (h *KubeHandler) CreatePod(c *gin.Context) {
	client, ok := h.getReadyClient(c)
	if !ok {
		return
	}
	namespace := c.DefaultQuery("namespace", "default")

	var req createPodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	pod, err := client.CreatePod(c.Request.Context(), namespace, domain.KubePodSpec{
		Name:      req.Name,
		Namespace: namespace,
		Image:     req.Image,
		Labels:    req.Labels,
	})
	if err != nil {
		_ = c.Error(response.Internal("create pod failed: " + err.Error()))
		return
	}
	response.Created(c, pod)
}

func (h *KubeHandler) DeletePod(c *gin.Context) {
	client, ok := h.getReadyClient(c)
	if !ok {
		return
	}
	namespace := c.DefaultQuery("namespace", "default")
	name := c.Param("name")

	if err := client.DeletePod(c.Request.Context(), namespace, name); err != nil {
		_ = c.Error(response.Internal("delete pod failed: " + err.Error()))
		return
	}
	response.NoContent(c)
}

func (h *KubeHandler) ListServices(c *gin.Context) {
	client, ok := h.getReadyClient(c)
	if !ok {
		return
	}
	namespace := c.DefaultQuery("namespace", "")
	services, err := client.ListServices(c.Request.Context(), namespace)
	if err != nil {
		_ = c.Error(response.Internal("list services failed: " + err.Error()))
		return
	}
	response.OK(c, services)
}

func (h *KubeHandler) CreateService(c *gin.Context) {
	client, ok := h.getReadyClient(c)
	if !ok {
		return
	}
	namespace := c.DefaultQuery("namespace", "default")

	var req createServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	svc, err := client.CreateService(c.Request.Context(), namespace, domain.KubeServiceSpec{
		Name:      req.Name,
		Namespace: namespace,
		Type:      req.Type,
		Selector:  req.Selector,
		Ports:     req.Ports,
	})
	if err != nil {
		_ = c.Error(response.Internal("create service failed: " + err.Error()))
		return
	}
	response.Created(c, svc)
}

func (h *KubeHandler) DeleteService(c *gin.Context) {
	client, ok := h.getReadyClient(c)
	if !ok {
		return
	}
	namespace := c.DefaultQuery("namespace", "default")
	name := c.Param("name")

	if err := client.DeleteService(c.Request.Context(), namespace, name); err != nil {
		_ = c.Error(response.Internal("delete service failed: " + err.Error()))
		return
	}
	response.NoContent(c)
}

func (h *KubeHandler) ListPersistentVolumes(c *gin.Context) {
	client, ok := h.getReadyClient(c)
	if !ok {
		return
	}
	pvs, err := client.ListPersistentVolumes(c.Request.Context())
	if err != nil {
		_ = c.Error(response.Internal("list persistent volumes failed: " + err.Error()))
		return
	}
	response.OK(c, pvs)
}

func (h *KubeHandler) ListPersistentVolumeClaims(c *gin.Context) {
	client, ok := h.getReadyClient(c)
	if !ok {
		return
	}
	namespace := c.DefaultQuery("namespace", "")
	pvcs, err := client.ListPersistentVolumeClaims(c.Request.Context(), namespace)
	if err != nil {
		_ = c.Error(response.Internal("list persistent volume claims failed: " + err.Error()))
		return
	}
	response.OK(c, pvcs)
}
