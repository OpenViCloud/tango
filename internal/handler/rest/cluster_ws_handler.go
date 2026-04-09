package rest

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ClusterLogStreamer is implemented by ansible.LogBroadcaster.
type ClusterLogStreamer interface {
	Subscribe(clusterID string) (<-chan []byte, func())
}

type ClusterWSHandler struct {
	streamer ClusterLogStreamer
}

func NewClusterWSHandler(streamer ClusterLogStreamer) *ClusterWSHandler {
	return &ClusterWSHandler{streamer: streamer}
}

func (h *ClusterWSHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/ws/clusters/:id/logs", h.StreamLogs)
}

func (h *ClusterWSHandler) StreamLogs(c *gin.Context) {
	clusterID := c.Param("id")

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

	ch, unsub := h.streamer.Subscribe(clusterID)
	if ch == nil {
		// No active provisioning — cluster is already in a terminal state
		_ = send(wsMsg{T: "done", Status: "no active provisioning"})
		return
	}
	defer unsub()

	for line := range ch {
		if err := send(wsMsg{T: "log", D: string(line)}); err != nil {
			return // client disconnected
		}
	}

	_ = send(wsMsg{T: "done"})
	_ = conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
