package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"tango/internal/application/query"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// BuildLogStreamer is implemented by BuildService to stream live log chunks.
type BuildLogStreamer interface {
	Subscribe(jobID string) (<-chan []byte, func())
}

type BuildWSHandler struct {
	streamer BuildLogStreamer
	getByID  *query.GetBuildJobHandler
}

func NewBuildWSHandler(streamer BuildLogStreamer, getByID *query.GetBuildJobHandler) *BuildWSHandler {
	return &BuildWSHandler{streamer: streamer, getByID: getByID}
}

func (h *BuildWSHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/ws/builds/:id/logs", h.StreamLogs)
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

const wsPingInterval = 30 * time.Second

// wsPingLoop sends a ping every 30 s to keep the connection alive through
// NAT/proxies. It stops when done is closed or a ping write fails.
func wsPingLoop(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		}
	}
}

type wsMsg struct {
	T        string `json:"t"`
	D        string `json:"d,omitempty"`
	Status   string `json:"status,omitempty"`
	ID       string `json:"id,omitempty"`
	Progress string `json:"progress,omitempty"`
}

func (h *BuildWSHandler) StreamLogs(c *gin.Context) {
	jobID := c.Param("id")

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // upgrade writes its own HTTP error
	}
	defer conn.Close()

	pingDone := make(chan struct{})
	defer close(pingDone)
	go wsPingLoop(conn, pingDone)

	send := func(msg wsMsg) error {
		data, _ := json.Marshal(msg)
		return conn.WriteMessage(websocket.TextMessage, data)
	}

	ch, unsub := h.streamer.Subscribe(jobID)

	if ch == nil {
		// Build not running — fetch stored logs from DB
		job, err := h.getByID.Handle(c.Request.Context(), jobID)
		if err != nil {
			_ = send(wsMsg{T: "error", D: "job not found"})
			return
		}
		if job.Logs != "" {
			_ = send(wsMsg{T: "log", D: job.Logs})
		}
		_ = send(wsMsg{T: "done", Status: string(job.Status)})
		return
	}
	defer unsub()

	// Drain the live channel until it's closed (build done).
	for chunk := range ch {
		if err := send(wsMsg{T: "log", D: string(chunk)}); err != nil {
			return // client disconnected
		}
	}

	// Channel closed → build finished. Fetch final status from DB.
	job, err := h.getByID.Handle(c.Request.Context(), jobID)
	status := "done"
	if err == nil {
		status = string(job.Status)
	}
	_ = send(wsMsg{T: "done", Status: status})
	_ = conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
