package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type LogOutputMode string
type LogFormat string

const (
	LogOutputStdout LogOutputMode = "stdout"
	LogOutputFile   LogOutputMode = "file"
	LogOutputBoth   LogOutputMode = "both"

	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"

	defaultTailLines = 200
	maxTailLines     = 1000
)

type LogConfig struct {
	Level      slog.Level
	Format     LogFormat
	Output     LogOutputMode
	FilePath   string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

type logService struct {
	logger   *slog.Logger
	filePath string
}

func NewLogService(cfg LogConfig) (appservices.LogService, error) {
	writer, filePath, err := buildLogWriter(cfg)
	if err != nil {
		return nil, err
	}
	handler := newLogHandler(writer, cfg)
	logger := slog.New(handler)
	return &logService{
		logger:   logger,
		filePath: filePath,
	}, nil
}

func (s *logService) Logger() *slog.Logger {
	if s == nil || s.logger == nil {
		return slog.Default()
	}
	return s.logger
}

func (s *logService) Tail(_ context.Context, req appservices.TailLogsRequest) (*appservices.TailLogsResult, error) {
	if s == nil || strings.TrimSpace(s.filePath) == "" {
		return nil, fmt.Errorf("log file output is not configured: %w", domain.ErrInvalidInput)
	}
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &appservices.TailLogsResult{
				FilePath: s.filePath,
				Lines:    nil,
			}, nil
		}
		return nil, fmt.Errorf("read log file: %w", err)
	}
	lines := splitLogLines(string(data))
	lines = filterLogLines(lines, req)
	limit := req.Lines
	if limit <= 0 {
		limit = defaultTailLines
	}
	if limit > maxTailLines {
		limit = maxTailLines
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return &appservices.TailLogsResult{
		FilePath: s.filePath,
		Lines:    lines,
		Items:    parseLogEntries(lines),
	}, nil
}

func filterLogLines(lines []string, req appservices.TailLogsRequest) []string {
	traceID := strings.TrimSpace(req.TraceID)
	date := strings.TrimSpace(req.Date)
	level := strings.ToUpper(strings.TrimSpace(req.Level))
	contains := strings.ToLower(strings.TrimSpace(req.Contains))

	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if traceID != "" && !strings.Contains(line, traceID) {
			continue
		}
		if date != "" && !matchesLogDate(line, date) {
			continue
		}
		if level != "" && !matchesLogLevel(line, level) {
			continue
		}
		if contains != "" && !strings.Contains(strings.ToLower(line), contains) {
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered
}

func matchesLogDate(line string, date string) bool {
	return strings.Contains(line, "time="+date) || strings.Contains(line, "\"time\":\""+date)
}

func matchesLogLevel(line string, level string) bool {
	return strings.Contains(line, "level="+level) || strings.Contains(line, "\"level\":\""+level+"\"")
}

func buildLogWriter(cfg LogConfig) (io.Writer, string, error) {
	output := normalizeLogOutput(cfg.Output)
	filePath := strings.TrimSpace(cfg.FilePath)
	switch output {
	case LogOutputStdout:
		return os.Stdout, "", nil
	case LogOutputFile:
		fileWriter, cleanPath, err := newRotatingFileWriter(cfg, filePath)
		if err != nil {
			return nil, "", err
		}
		return fileWriter, cleanPath, nil
	case LogOutputBoth:
		fileWriter, cleanPath, err := newRotatingFileWriter(cfg, filePath)
		if err != nil {
			return nil, "", err
		}
		return io.MultiWriter(os.Stdout, fileWriter), cleanPath, nil
	default:
		return nil, "", fmt.Errorf("unsupported log output %q: %w", cfg.Output, domain.ErrInvalidInput)
	}
}

func newRotatingFileWriter(cfg LogConfig, path string) (io.Writer, string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, "", fmt.Errorf("log file path is required: %w", domain.ErrInvalidInput)
	}
	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, "", fmt.Errorf("create log directory: %w", err)
	}
	return &lumberjack.Logger{
		Filename:   cleanPath,
		MaxSize:    positiveOrDefault(cfg.MaxSizeMB, 20),
		MaxBackups: positiveOrDefault(cfg.MaxBackups, 10),
		MaxAge:     positiveOrDefault(cfg.MaxAgeDays, 7),
		Compress:   cfg.Compress,
	}, cleanPath, nil
}

func newLogHandler(writer io.Writer, cfg LogConfig) slog.Handler {
	options := &slog.HandlerOptions{
		Level: cfg.Level,
	}
	switch normalizeLogFormat(cfg.Format) {
	case LogFormatJSON:
		return slog.NewJSONHandler(writer, options)
	default:
		return slog.NewTextHandler(writer, options)
	}
}

func normalizeLogOutput(output LogOutputMode) LogOutputMode {
	switch LogOutputMode(strings.ToLower(strings.TrimSpace(string(output)))) {
	case LogOutputFile:
		return LogOutputFile
	case LogOutputBoth:
		return LogOutputBoth
	default:
		return LogOutputStdout
	}
}

func normalizeLogFormat(format LogFormat) LogFormat {
	switch LogFormat(strings.ToLower(strings.TrimSpace(string(format)))) {
	case LogFormatJSON:
		return LogFormatJSON
	default:
		return LogFormatText
	}
}

func positiveOrDefault(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func splitLogLines(raw string) []string {
	items := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
}

func parseLogEntries(lines []string) []appservices.LogEntry {
	items := make([]appservices.LogEntry, 0, len(lines))
	for _, line := range lines {
		items = append(items, parseLogEntry(line))
	}
	return items
}

func parseLogEntry(line string) appservices.LogEntry {
	entry := appservices.LogEntry{
		Raw:    line,
		Fields: map[string]string{},
	}
	for _, token := range tokenizeLogLine(line) {
		key, value, ok := strings.Cut(token, "=")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"`)
		entry.Fields[key] = value
		switch key {
		case "time":
			entry.Time = value
		case "level":
			entry.Level = value
		case "msg":
			entry.Message = value
		case "traceId":
			entry.TraceID = value
		case "method":
			entry.Method = value
		case "path":
			entry.Path = value
		case "query":
			entry.Query = value
		case "client_ip":
			entry.ClientIP = value
		case "status":
			if status, err := strconv.Atoi(value); err == nil {
				entry.Status = status
			}
		}
	}
	return entry
}

func tokenizeLogLine(line string) []string {
	tokens := make([]string, 0, 16)
	var current strings.Builder
	inQuotes := false
	escaped := false
	for _, r := range line {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			current.WriteRune(r)
			escaped = true
		case r == '"':
			current.WriteRune(r)
			inQuotes = !inQuotes
		case r == ' ' && !inQuotes:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}
