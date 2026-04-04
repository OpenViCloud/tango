package rest

import (
	"net/http"

	"tango/internal/domain"

	"github.com/gin-gonic/gin"
)

// SwarmHandler exposes cluster-related endpoints.
// When swarmRepo is nil (Docker unavailable) every endpoint returns a
// safe "not in cluster" response so the frontend degrades gracefully.
type SwarmHandler struct {
	swarmRepo domain.SwarmRepository
}

func NewSwarmHandler(swarmRepo domain.SwarmRepository) *SwarmHandler {
	return &SwarmHandler{swarmRepo: swarmRepo}
}

func (h *SwarmHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/swarm/status", h.GetStatus)
	rg.GET("/swarm/nodes", h.ListNodes)
}

type swarmStatusResponse struct {
	IsManager bool `json:"is_manager"`
}

type swarmNodeResponse struct {
	ID           string `json:"id"`
	Hostname     string `json:"hostname"`
	Role         string `json:"role"`
	State        string `json:"state"`
	Availability string `json:"availability"`
	ManagerAddr  string `json:"manager_addr,omitempty"`
}

// GetStatus returns whether the local Docker daemon is a swarm manager.
func (h *SwarmHandler) GetStatus(c *gin.Context) {
	if h.swarmRepo == nil {
		c.JSON(http.StatusOK, swarmStatusResponse{IsManager: false})
		return
	}
	c.JSON(http.StatusOK, swarmStatusResponse{
		IsManager: h.swarmRepo.IsManager(c.Request.Context()),
	})
}

// ListNodes returns all nodes in the swarm cluster.
// Returns 503 when not a swarm manager.
func (h *SwarmHandler) ListNodes(c *gin.Context) {
	if h.swarmRepo == nil || !h.swarmRepo.IsManager(c.Request.Context()) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "not a swarm manager"})
		return
	}
	nodes, err := h.swarmRepo.ListNodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]swarmNodeResponse, 0, len(nodes))
	for _, n := range nodes {
		resp = append(resp, swarmNodeResponse{
			ID:           n.ID,
			Hostname:     n.Hostname,
			Role:         n.Role,
			State:        n.State,
			Availability: n.Availability,
			ManagerAddr:  n.ManagerAddr,
		})
	}
	c.JSON(http.StatusOK, resp)
}
