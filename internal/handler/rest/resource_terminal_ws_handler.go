package rest

import (
	"context"
	"encoding/json"
	"io"
	"strconv"

	"tango/internal/application/query"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ResourceTerminalWSHandler struct {
	dockerRepo  domain.DockerRepository
	getResource *query.GetResourceHandler
}

func NewResourceTerminalWSHandler(dockerRepo domain.DockerRepository, getResource *query.GetResourceHandler) *ResourceTerminalWSHandler {
	return &ResourceTerminalWSHandler{dockerRepo: dockerRepo, getResource: getResource}
}

func (h *ResourceTerminalWSHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/ws/resources/:resourceId/terminal", h.Connect)
}

type terminalClientMsg struct {
	T    string `json:"t"`
	D    string `json:"d,omitempty"`
	Cols uint   `json:"cols,omitempty"`
	Rows uint   `json:"rows,omitempty"`
}

func (h *ResourceTerminalWSHandler) Connect(c *gin.Context) {
	resource, err := h.getResource.Handle(c.Request.Context(), query.GetResourceQuery{ID: c.Param("resourceId")})
	if err != nil {
		_ = c.Error(response.NotFound(err.Error()))
		return
	}
	if h.dockerRepo == nil {
		_ = c.Error(response.BadRequest("docker is unavailable"))
		return
	}
	if resource.ContainerID == "" || resource.Status != domain.ResourceStatusRunning {
		_ = c.Error(response.BadRequest("resource is not running"))
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	pingDone := make(chan struct{})
	defer close(pingDone)
	go wsPingLoop(conn, pingDone)

	cols := parseUintDefault(c.Query("cols"), 120)
	rows := parseUintDefault(c.Query("rows"), 30)
	session, err := h.dockerRepo.ExecContainer(c.Request.Context(), resource.ContainerID, domain.ContainerExecInput{
		Shell: []string{"/bin/bash", "/bin/sh"},
		Cols:  cols,
		Rows:  rows,
	})
	if err != nil {
		_ = sendTerminalError(conn, err.Error())
		return
	}
	defer session.Close()

	done := make(chan struct{})
	defer close(done)

	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := session.Read(buffer)
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buffer[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					_ = sendTerminalError(conn, err.Error())
				}
				_ = conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			select {
			case <-done:
				return
			default:
			}
		}
	}()

	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if msgType != websocket.TextMessage {
			continue
		}

		var msg terminalClientMsg
		if err := json.Unmarshal(payload, &msg); err != nil {
			_ = sendTerminalError(conn, "invalid terminal message")
			continue
		}

		switch msg.T {
		case "input":
			if msg.D == "" {
				continue
			}
			if _, err := io.WriteString(session, msg.D); err != nil {
				_ = sendTerminalError(conn, err.Error())
				return
			}
		case "resize":
			if msg.Cols == 0 || msg.Rows == 0 {
				continue
			}
			if err := session.Resize(context.Background(), msg.Cols, msg.Rows); err != nil {
				_ = sendTerminalError(conn, err.Error())
				return
			}
		}
	}
}

func sendTerminalError(conn *websocket.Conn, message string) error {
	data, _ := json.Marshal(wsMsg{T: "error", D: message})
	return conn.WriteMessage(websocket.TextMessage, data)
}

func parseUintDefault(value string, fallback uint) uint {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fallback
	}
	return uint(parsed)
}
