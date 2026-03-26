package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SuccessEnvelope[T any] struct {
	Success bool   `json:"success"`
	TraceID string `json:"traceId"`
	Data    T      `json:"data"`
}

func OK[T any](c *gin.Context, data T) {
	c.JSON(http.StatusOK, SuccessEnvelope[T]{
		Success: true,
		TraceID: TraceID(c),
		Data:    data,
	})
}

func Created[T any](c *gin.Context, data T) {
	c.JSON(http.StatusCreated, SuccessEnvelope[T]{
		Success: true,
		TraceID: TraceID(c),
		Data:    data,
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
