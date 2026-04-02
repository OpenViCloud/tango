package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	nethttp "net/http"
	"os"
	"strconv"
	"strings"

	"tango/internal/runner/model"
	"tango/internal/runner/service"
)

type Handler struct {
	token          string
	mysqlRunner    *service.MySQLRunner
	mariadbRunner  *service.MariaDBRunner
	postgresRunner *service.PostgresRunner
	mongoRunner    *service.MongoRunner
}

func NewHandler(token string, mysqlRunner *service.MySQLRunner, mariadbRunner *service.MariaDBRunner, postgresRunner *service.PostgresRunner, mongoRunner *service.MongoRunner) *Handler {
	return &Handler{
		token:          strings.TrimSpace(token),
		mysqlRunner:    mysqlRunner,
		mariadbRunner:  mariadbRunner,
		postgresRunner: postgresRunner,
		mongoRunner:    mongoRunner,
	}
}

func (h *Handler) Healthz(w nethttp.ResponseWriter, _ *nethttp.Request) {
	w.WriteHeader(nethttp.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) MySQLLogicalDump(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !h.authorize(w, r) {
		return
	}
	var req model.MySQLLogicalDumpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("decode mysql dump request: %w", err))
		return
	}
	tempFile, err := os.CreateTemp("", "tango-runner-mysql-dump-*")
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("create temp dump file: %w", err))
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	artifactName, err := h.mysqlRunner.RunLogicalDump(r.Context(), &req, tempFile)
	closeErr := tempFile.Close()
	if err != nil {
		slog.Default().Error("runner mysql logical dump failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	if closeErr != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("close temp dump file: %w", closeErr))
		return
	}
	file, err := os.Open(tempPath)
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("open temp dump file: %w", err))
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Tango-Artifact-File-Name", artifactName)
	w.Header().Set("X-Tango-MySQL-Version", strings.TrimSpace(req.Version))
	w.Header().Set("X-Tango-Compression-Type", strings.TrimSpace(req.CompressionType))
	if _, err := io.Copy(w, file); err != nil {
		slog.Default().Error("runner mysql logical dump stream failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
	}
}

func (h *Handler) MySQLLogicalRestore(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !h.authorize(w, r) {
		return
	}
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("parse multipart form: %w", err))
		return
	}
	req, err := decodeRestoreRequest(r.MultipartForm)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err)
		return
	}
	file, _, err := r.FormFile("artifact")
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("read restore artifact: %w", err))
		return
	}
	defer file.Close()
	if err := h.mysqlRunner.RunLogicalRestore(r.Context(), req, file); err != nil {
		slog.Default().Error("runner mysql logical restore failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *Handler) MariaDBLogicalDump(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !h.authorize(w, r) {
		return
	}
	var req model.MariaDBLogicalDumpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("decode mariadb dump request: %w", err))
		return
	}
	tempFile, err := os.CreateTemp("", "tango-runner-mariadb-dump-*")
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("create temp dump file: %w", err))
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	artifactName, err := h.mariadbRunner.RunLogicalDump(r.Context(), &req, tempFile)
	closeErr := tempFile.Close()
	if err != nil {
		slog.Default().Error("runner mariadb logical dump failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	if closeErr != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("close temp dump file: %w", closeErr))
		return
	}
	file, err := os.Open(tempPath)
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("open temp dump file: %w", err))
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Tango-Artifact-File-Name", artifactName)
	w.Header().Set("X-Tango-MariaDB-Version", strings.TrimSpace(req.Version))
	w.Header().Set("X-Tango-Compression-Type", strings.TrimSpace(req.CompressionType))
	if _, err := io.Copy(w, file); err != nil {
		slog.Default().Error("runner mariadb logical dump stream failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
	}
}

func (h *Handler) MariaDBLogicalRestore(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !h.authorize(w, r) {
		return
	}
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("parse multipart form: %w", err))
		return
	}
	req, err := decodeMariaDBRestoreRequest(r.MultipartForm)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err)
		return
	}
	file, _, err := r.FormFile("artifact")
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("read restore artifact: %w", err))
		return
	}
	defer file.Close()
	if err := h.mariadbRunner.RunLogicalRestore(r.Context(), req, file); err != nil {
		slog.Default().Error("runner mariadb logical restore failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *Handler) MongoLogicalDump(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !h.authorize(w, r) {
		return
	}
	var req model.MongoLogicalDumpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("decode mongo dump request: %w", err))
		return
	}
	tempFile, err := os.CreateTemp("", "tango-runner-mongo-dump-*")
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("create temp dump file: %w", err))
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	artifactName, err := h.mongoRunner.RunLogicalDump(r.Context(), &req, tempFile)
	closeErr := tempFile.Close()
	if err != nil {
		slog.Default().Error("runner mongo logical dump failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	if closeErr != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("close temp dump file: %w", closeErr))
		return
	}
	file, err := os.Open(tempPath)
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("open temp dump file: %w", err))
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Tango-Artifact-File-Name", artifactName)
	w.Header().Set("X-Tango-Compression-Type", strings.TrimSpace(req.CompressionType))
	w.Header().Set("X-Tango-Mongo-Database", strings.TrimSpace(req.Database))
	if _, err := io.Copy(w, file); err != nil {
		slog.Default().Error("runner mongo logical dump stream failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
	}
}

func (h *Handler) MongoLogicalRestore(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !h.authorize(w, r) {
		return
	}
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("parse multipart form: %w", err))
		return
	}
	req, err := decodeMongoRestoreRequest(r.MultipartForm)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err)
		return
	}
	file, _, err := r.FormFile("artifact")
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("read restore artifact: %w", err))
		return
	}
	defer file.Close()
	if err := h.mongoRunner.RunLogicalRestore(r.Context(), req, file); err != nil {
		slog.Default().Error("runner mongo logical restore failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *Handler) PostgresLogicalDump(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !h.authorize(w, r) {
		return
	}
	var req model.PostgresLogicalDumpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("decode postgres dump request: %w", err))
		return
	}
	tempFile, err := os.CreateTemp("", "tango-runner-postgres-dump-*")
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("create temp dump file: %w", err))
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	artifactName, err := h.postgresRunner.RunLogicalDump(r.Context(), &req, tempFile)
	closeErr := tempFile.Close()
	if err != nil {
		slog.Default().Error("runner postgres logical dump failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	if closeErr != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("close temp dump file: %w", closeErr))
		return
	}
	file, err := os.Open(tempPath)
	if err != nil {
		writeError(w, nethttp.StatusInternalServerError, fmt.Errorf("open temp dump file: %w", err))
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Tango-Artifact-File-Name", artifactName)
	w.Header().Set("X-Tango-Postgres-Version", strings.TrimSpace(req.Version))
	w.Header().Set("X-Tango-Compression-Type", strings.TrimSpace(req.CompressionType))
	w.Header().Set("X-Tango-Postgres-Format", "custom")
	if _, err := io.Copy(w, file); err != nil {
		slog.Default().Error("runner postgres logical dump stream failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
	}
}

func (h *Handler) PostgresLogicalRestore(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !h.authorize(w, r) {
		return
	}
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("parse multipart form: %w", err))
		return
	}
	req, err := decodePostgresRestoreRequest(r.MultipartForm)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, err)
		return
	}
	file, _, err := r.FormFile("artifact")
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, fmt.Errorf("read restore artifact: %w", err))
		return
	}
	defer file.Close()
	if err := h.postgresRunner.RunLogicalRestore(r.Context(), req, file); err != nil {
		slog.Default().Error("runner postgres logical restore failed", "err", err, "database", req.Database, "host", req.Host, "port", req.Port)
		writeError(w, nethttp.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *Handler) authorize(w nethttp.ResponseWriter, r *nethttp.Request) bool {
	if h.token == "" {
		return true
	}
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	expected := "Bearer " + h.token
	if authz != expected {
		writeError(w, nethttp.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return false
	}
	return true
}

func decodeRestoreRequest(form *multipart.Form) (*model.MySQLLogicalDumpRequest, error) {
	port, err := strconv.Atoi(strings.TrimSpace(firstFormValue(form, "port")))
	if err != nil {
		return nil, fmt.Errorf("invalid restore port: %w", err)
	}
	return &model.MySQLLogicalDumpRequest{
		Version:         firstFormValue(form, "version"),
		Host:            firstFormValue(form, "host"),
		Port:            port,
		Username:        firstFormValue(form, "username"),
		Password:        firstFormValue(form, "password"),
		Database:        firstFormValue(form, "database"),
		CompressionType: firstFormValue(form, "compression_type"),
	}, nil
}

func decodeMongoRestoreRequest(form *multipart.Form) (*model.MongoLogicalRestoreRequest, error) {
	port, err := strconv.Atoi(strings.TrimSpace(firstFormValue(form, "port")))
	if err != nil {
		return nil, fmt.Errorf("invalid restore port: %w", err)
	}
	return &model.MongoLogicalRestoreRequest{
		Host:            firstFormValue(form, "host"),
		Port:            port,
		Username:        firstFormValue(form, "username"),
		Password:        firstFormValue(form, "password"),
		Database:        firstFormValue(form, "database"),
		AuthDatabase:    firstFormValue(form, "auth_database"),
		ConnectionURI:   firstFormValue(form, "connection_uri"),
		SourceDatabase:  firstFormValue(form, "source_database"),
		CompressionType: firstFormValue(form, "compression_type"),
	}, nil
}

func decodeMariaDBRestoreRequest(form *multipart.Form) (*model.MariaDBLogicalDumpRequest, error) {
	port, err := strconv.Atoi(strings.TrimSpace(firstFormValue(form, "port")))
	if err != nil {
		return nil, fmt.Errorf("invalid restore port: %w", err)
	}
	return &model.MariaDBLogicalDumpRequest{
		Version:         firstFormValue(form, "version"),
		Host:            firstFormValue(form, "host"),
		Port:            port,
		Username:        firstFormValue(form, "username"),
		Password:        firstFormValue(form, "password"),
		Database:        firstFormValue(form, "database"),
		CompressionType: firstFormValue(form, "compression_type"),
	}, nil
}

func decodePostgresRestoreRequest(form *multipart.Form) (*model.PostgresLogicalDumpRequest, error) {
	port, err := strconv.Atoi(strings.TrimSpace(firstFormValue(form, "port")))
	if err != nil {
		return nil, fmt.Errorf("invalid restore port: %w", err)
	}
	return &model.PostgresLogicalDumpRequest{
		Version:         firstFormValue(form, "version"),
		Host:            firstFormValue(form, "host"),
		Port:            port,
		Username:        firstFormValue(form, "username"),
		Password:        firstFormValue(form, "password"),
		Database:        firstFormValue(form, "database"),
		CompressionType: firstFormValue(form, "compression_type"),
	}, nil
}

func firstFormValue(form *multipart.Form, key string) string {
	values := form.Value[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func writeError(w nethttp.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.ErrorResponse{Error: err.Error()})
}

func buildDumpFileName(database string, compressionType string) string {
	name := strings.TrimSpace(database)
	if name == "" {
		name = "mysql-backup"
	}
	if strings.EqualFold(compressionType, "gzip") {
		return name + ".sql.gz"
	}
	return name + ".sql"
}

func buildMongoDumpFileName(database string, compressionType string) string {
	name := strings.TrimSpace(database)
	if name == "" {
		name = "mongodb-backup"
	}
	if strings.EqualFold(compressionType, "gzip") {
		return name + ".archive.gz"
	}
	return name + ".archive"
}

func buildPostgresDumpFileName(database string, compressionType string) string {
	name := strings.TrimSpace(database)
	if name == "" {
		name = "postgres-backup"
	}
	if strings.EqualFold(compressionType, "gzip") {
		return name + ".dump.gz"
	}
	return name + ".dump"
}
