package rest

import (
	"context"
	"net/http"
	"time"

	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SSHManager is the subset of ssh.Manager needed by ServerHandler.
type SSHManager interface {
	EnsureKeypair(ctx context.Context) error
	PublicKey(ctx context.Context) (string, error)
	Ping(ctx context.Context, server *domain.Server) error
}

type ServerHandler struct {
	serverRepo domain.ServerRepository
	sshManager SSHManager
}

func NewServerHandler(serverRepo domain.ServerRepository, sshManager SSHManager) *ServerHandler {
	return &ServerHandler{serverRepo: serverRepo, sshManager: sshManager}
}

func (h *ServerHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/servers/ssh-public-key", h.GetSSHPublicKey)
	rg.GET("/servers", h.List)
	rg.POST("/servers", h.Create)
	rg.GET("/servers/:id", h.Get)
	rg.DELETE("/servers/:id", h.Delete)
	rg.POST("/servers/:id/ping", h.Ping)
}

type createServerRequest struct {
	Name      string `json:"name" binding:"required"`
	PublicIP  string `json:"public_ip" binding:"required"`
	PrivateIP string `json:"private_ip"`
	SSHUser   string `json:"ssh_user"`
	SSHPort   int    `json:"ssh_port"`
}

type serverResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	PublicIP    string     `json:"public_ip"`
	PrivateIP   string     `json:"private_ip"`
	SSHUser     string     `json:"ssh_user"`
	SSHPort     int        `json:"ssh_port"`
	Status      string     `json:"status"`
	ErrorMsg    string     `json:"error_msg,omitempty"`
	LastPingAt  *time.Time `json:"last_ping_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func toServerResponse(s *domain.Server) serverResponse {
	return serverResponse{
		ID:         s.ID,
		Name:       s.Name,
		PublicIP:   s.PublicIP,
		PrivateIP:  s.PrivateIP,
		SSHUser:    s.SSHUser,
		SSHPort:    s.SSHPort,
		Status:     string(s.Status),
		ErrorMsg:   s.ErrorMsg,
		LastPingAt: s.LastPingAt,
		CreatedAt:  s.CreatedAt,
	}
}

func (h *ServerHandler) GetSSHPublicKey(c *gin.Context) {
	if err := h.sshManager.EnsureKeypair(c.Request.Context()); err != nil {
		_ = c.Error(response.Internal("ensure SSH keypair failed"))
		return
	}
	pub, err := h.sshManager.PublicKey(c.Request.Context())
	if err != nil {
		_ = c.Error(response.Internal("get SSH public key failed"))
		return
	}
	response.OK(c, gin.H{"public_key": pub})
}

func (h *ServerHandler) List(c *gin.Context) {
	servers, err := h.serverRepo.ListAll(c.Request.Context())
	if err != nil {
		_ = c.Error(response.Internal("list servers failed"))
		return
	}
	resp := make([]serverResponse, 0, len(servers))
	for _, s := range servers {
		resp = append(resp, toServerResponse(s))
	}
	response.OK(c, resp)
}

func (h *ServerHandler) Create(c *gin.Context) {
	var req createServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.BadRequest(err.Error()))
		return
	}

	sshUser := req.SSHUser
	if sshUser == "" {
		sshUser = "root"
	}
	sshPort := req.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	now := time.Now().UTC()
	server := &domain.Server{
		ID:        uuid.NewString(),
		Name:      req.Name,
		PublicIP:  req.PublicIP,
		PrivateIP: req.PrivateIP,
		SSHUser:   sshUser,
		SSHPort:   sshPort,
		Status:    domain.ServerStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	saved, err := h.serverRepo.Save(c.Request.Context(), server)
	if err != nil {
		_ = c.Error(response.Internal("save server failed"))
		return
	}
	response.Created(c, toServerResponse(saved))
}

func (h *ServerHandler) Get(c *gin.Context) {
	id := c.Param("id")
	server, err := h.serverRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrServerNotFound {
			_ = c.Error(response.NotFound("server not found"))
			return
		}
		_ = c.Error(response.Internal("get server failed"))
		return
	}
	response.OK(c, toServerResponse(server))
}

func (h *ServerHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.serverRepo.Delete(c.Request.Context(), id); err != nil {
		if err == domain.ErrServerNotFound {
			_ = c.Error(response.NotFound("server not found"))
			return
		}
		_ = c.Error(response.Internal("delete server failed"))
		return
	}
	response.NoContent(c)
}

func (h *ServerHandler) Ping(c *gin.Context) {
	id := c.Param("id")
	server, err := h.serverRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrServerNotFound {
			_ = c.Error(response.NotFound("server not found"))
			return
		}
		_ = c.Error(response.Internal("get server failed"))
		return
	}

	pingErr := h.sshManager.Ping(c.Request.Context(), server)

	now := time.Now().UTC()
	if pingErr != nil {
		server.Status = domain.ServerStatusError
		server.ErrorMsg = pingErr.Error()
	} else {
		server.Status = domain.ServerStatusConnected
		server.ErrorMsg = ""
		server.LastPingAt = &now
	}
	server.UpdatedAt = now

	updated, err := h.serverRepo.Update(c.Request.Context(), server)
	if err != nil {
		_ = c.Error(response.Internal("update server status failed"))
		return
	}

	if pingErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"status":  string(updated.Status),
			"error":   pingErr.Error(),
		})
		return
	}
	response.OK(c, gin.H{
		"status": string(updated.Status),
	})
}
