package services

import (
	"context"
	"log/slog"
)

type TailLogsRequest struct {
	Lines   int
	TraceID string
	Date    string
	Level   string
	Contains string
}

type TailLogsResult struct {
	FilePath string
	Lines    []string
	Items    []LogEntry
}

type LogEntry struct {
	Raw      string
	Time     string
	Level    string
	Message  string
	TraceID  string
	Method   string
	Path     string
	Status   int
	Query    string
	ClientIP string
	Fields   map[string]string
}

type LogService interface {
	Logger() *slog.Logger
	Tail(ctx context.Context, req TailLogsRequest) (*TailLogsResult, error)
}
