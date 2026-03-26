package response

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
)

type FieldError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorBody struct {
	Code    string                  `json:"code"`
	Message string                  `json:"message"`
	Details map[string][]FieldError `json:"details,omitempty"`
}

type ErrorEnvelope struct {
	Success bool      `json:"success"`
	TraceID string    `json:"traceId"`
	Error   ErrorBody `json:"error"`
}

type APIError struct {
	Status  int
	Code    string
	Message string
	Details map[string][]FieldError
	Cause   error
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func New(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

func Validation(details map[string][]FieldError, message string) *APIError {
	if message == "" {
		message = "Validation failed"
	}
	return &APIError{
		Status:  http.StatusUnprocessableEntity,
		Code:    "VALIDATION_ERROR",
		Message: message,
		Details: details,
	}
}

func BadRequest(message string) *APIError {
	if message == "" {
		message = "Bad request"
	}
	return New(http.StatusBadRequest, "BAD_REQUEST", message)
}

func Unauthorized(message string) *APIError {
	if message == "" {
		message = "Unauthorized"
	}
	return New(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(message string) *APIError {
	if message == "" {
		message = "Forbidden"
	}
	return New(http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(message string) *APIError {
	if message == "" {
		message = "Resource not found"
	}
	return New(http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(message string) *APIError {
	if message == "" {
		message = "Conflict"
	}
	return New(http.StatusConflict, "CONFLICT", message)
}

func Internal(message string) *APIError {
	if message == "" {
		message = "Internal server error"
	}
	return New(http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

func InternalCause(cause error, message string) *APIError {
	apiErr := Internal(message)
	apiErr.Cause = cause
	return apiErr
}

func newTraceID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("trace-%d", randFallback())
	}
	return hex.EncodeToString(b[:])
}

func randFallback() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	var out int64
	for _, part := range b {
		out = (out << 8) | int64(part)
	}
	if out < 0 {
		return -out
	}
	return out
}
