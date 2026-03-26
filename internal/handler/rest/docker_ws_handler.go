package rest

import (
	"bufio"
	"context"
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ImagePullStreamer is implemented by docker.Repository.
type ImagePullStreamer interface {
	PullImageStream(ctx context.Context, reference string) (io.ReadCloser, error)
}

type DockerWSHandler struct {
	puller ImagePullStreamer
}

func NewDockerWSHandler(puller ImagePullStreamer) *DockerWSHandler {
	return &DockerWSHandler{puller: puller}
}

func (h *DockerWSHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/ws/docker/images/pull", h.PullImage)
}

// pullEvent mirrors the NDJSON objects the Docker daemon emits during a pull.
type pullEvent struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Progress string `json:"progress,omitempty"`
	ID     string `json:"id,omitempty"`
}

func (h *DockerWSHandler) PullImage(c *gin.Context) {
	reference := c.Query("reference")
	if reference == "" {
		c.JSON(400, gin.H{"error": "reference is required"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	send := func(msg wsMsg) error {
		data, _ := json.Marshal(msg)
		return conn.WriteMessage(websocket.TextMessage, data)
	}

	stream, err := h.puller.PullImageStream(c.Request.Context(), reference)
	if err != nil {
		_ = send(wsMsg{T: "error", D: err.Error()})
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		var ev pullEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Error != "" {
			_ = send(wsMsg{T: "error", D: ev.Error})
			return
		}
		var msg wsMsg
		if ev.ID != "" {
			// Per-layer event — send structured so the UI can update each layer in place
			msg = wsMsg{T: "layer", ID: ev.ID, Status: ev.Status, Progress: ev.Progress}
		} else {
			msg = wsMsg{T: "log", D: ev.Status + "\n"}
		}
		if err := send(msg); err != nil {
			return
		}
	}

	_ = send(wsMsg{T: "done", Status: "pulled"})
}
