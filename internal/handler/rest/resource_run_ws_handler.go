package rest

import (
	"encoding/json"

	"tango/internal/application/query"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ResourceRunLogStreamer interface {
	Subscribe(runID string) (<-chan []byte, func())
}

type ResourceRunWSHandler struct {
	streamer ResourceRunLogStreamer
	getByID  *query.GetResourceRunHandler
}

func NewResourceRunWSHandler(streamer ResourceRunLogStreamer, getByID *query.GetResourceRunHandler) *ResourceRunWSHandler {
	return &ResourceRunWSHandler{streamer: streamer, getByID: getByID}
}

func (h *ResourceRunWSHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/ws/resource-runs/:id/logs", h.StreamLogs)
}

func (h *ResourceRunWSHandler) StreamLogs(c *gin.Context) {
	runID := c.Param("id")

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	pingDone := make(chan struct{})
	defer close(pingDone)
	go wsPingLoop(conn, pingDone)

	send := func(msg wsMsg) error {
		data, _ := json.Marshal(msg)
		return conn.WriteMessage(websocket.TextMessage, data)
	}

	ch, unsub := h.streamer.Subscribe(runID)
	if ch == nil {
		run, err := h.getByID.Handle(c.Request.Context(), runID)
		if err != nil {
			_ = send(wsMsg{T: "error", D: "run not found"})
			return
		}
		if run.Logs != "" {
			_ = send(wsMsg{T: "log", D: run.Logs})
		}
		_ = send(wsMsg{T: "done", Status: string(run.Status)})
		return
	}
	defer unsub()

	for chunk := range ch {
		if err := send(wsMsg{T: "log", D: string(chunk)}); err != nil {
			return
		}
	}

	run, err := h.getByID.Handle(c.Request.Context(), runID)
	status := "done"
	if err == nil {
		status = string(run.Status)
	}
	_ = send(wsMsg{T: "done", Status: status})
	_ = conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
