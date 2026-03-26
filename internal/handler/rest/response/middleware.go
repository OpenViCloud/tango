package response

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

const traceIDKey = "trace_id"

func Middleware(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(c *gin.Context) {
		traceID := newTraceID()
		c.Set(traceIDKey, traceID)
		c.Writer.Header().Set("X-Trace-Id", traceID)

		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", "traceId", traceID, "recovered", recovered)
				writeError(c, traceID, Internal(""))
				c.Abort()
			}
		}()

		c.Next()

		if c.Writer.Written() || len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			logger.Error("unhandled error", "traceId", traceID, "err", err)
			apiErr = Internal("")
		} else {
			logAPIError(logger, c, traceID, apiErr)
		}
		writeError(c, traceID, apiErr)
	}
}

func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		logger.Info("http request",
			"traceId", TraceID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(startedAt).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

func TraceID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if value, ok := c.Get(traceIDKey); ok {
		if traceID, ok := value.(string); ok {
			return traceID
		}
	}
	return ""
}

func writeError(c *gin.Context, traceID string, apiErr *APIError) {
	if apiErr == nil {
		apiErr = Internal("")
	}

	c.AbortWithStatusJSON(apiErr.Status, ErrorEnvelope{
		Success: false,
		TraceID: traceID,
		Error: ErrorBody{
			Code:    apiErr.Code,
			Message: apiErr.Message,
			Details: apiErr.Details,
		},
	})
}

func logAPIError(logger *slog.Logger, c *gin.Context, traceID string, apiErr *APIError) {
	if logger == nil || apiErr == nil || c == nil {
		return
	}
	args := []any{
		"traceId", traceID,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", apiErr.Status,
		"code", apiErr.Code,
	}
	if apiErr.Cause != nil {
		args = append(args, "cause", apiErr.Cause)
	}
	if apiErr.Status >= 500 {
		logger.Error("server error", args...)
		return
	}
	if apiErr.Status >= 400 {
		logger.Warn("client error", args...)
	}
}
