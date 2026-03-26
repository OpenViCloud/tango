package rest

import (
	"tango/internal/application/services"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	logs services.LogService
}

type tailLogsResponse struct {
	FilePath string   `json:"file_path"`
	Lines    []string `json:"lines"`
	Items    []logEntryResponse `json:"items"`
}

type logEntryResponse struct {
	Raw      string            `json:"raw"`
	Time     string            `json:"time,omitempty"`
	Level    string            `json:"level,omitempty"`
	Message  string            `json:"message,omitempty"`
	TraceID  string            `json:"trace_id,omitempty"`
	Method   string            `json:"method,omitempty"`
	Path     string            `json:"path,omitempty"`
	Status   int               `json:"status,omitempty"`
	Query    string            `json:"query,omitempty"`
	ClientIP string            `json:"client_ip,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"`
}

func NewLogHandler(logs services.LogService) *LogHandler {
	return &LogHandler{logs: logs}
}

func (h *LogHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/runtime/logs", h.Tail)
}

// Tail godoc
// @Summary Tail runtime logs
// @Tags system
// @Produce json
// @Security BearerAuth
// @Param lines query int false "Maximum lines to return"
// @Param traceId query string false "Optional trace ID filter"
// @Param date query string false "Optional date filter in YYYY-MM-DD"
// @Param level query string false "Optional level filter: error, warn, info, debug"
// @Param contains query string false "Optional substring filter"
// @Success 200 {object} tailLogsResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /runtime/logs [get]
func (h *LogHandler) Tail(c *gin.Context) {
	if h.logs == nil {
		_ = c.Error(response.Internal("log service is not initialized"))
		return
	}
	result, err := h.logs.Tail(c.Request.Context(), services.TailLogsRequest{
		Lines:    parseIntDefault(c.Query("lines"), 200),
		TraceID:  c.Query("traceId"),
		Date:     c.Query("date"),
		Level:    c.Query("level"),
		Contains: c.Query("contains"),
	})
	if err != nil {
		writeChannelError(c, err)
		return
	}
	response.OK(c, tailLogsResponse{
		FilePath: result.FilePath,
		Lines:    result.Lines,
		Items:    toLogEntryResponses(result.Items),
	})
}

func toLogEntryResponses(items []services.LogEntry) []logEntryResponse {
	if len(items) == 0 {
		return nil
	}
	resp := make([]logEntryResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, logEntryResponse{
			Raw:      item.Raw,
			Time:     item.Time,
			Level:    item.Level,
			Message:  item.Message,
			TraceID:  item.TraceID,
			Method:   item.Method,
			Path:     item.Path,
			Status:   item.Status,
			Query:    item.Query,
			ClientIP: item.ClientIP,
			Fields:   item.Fields,
		})
	}
	return resp
}
