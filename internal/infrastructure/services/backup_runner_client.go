package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

const (
	mySQLRunnerDumpPath       = "/internal/mysql/logical-dump"
	mySQLRunnerRestorePath    = "/internal/mysql/logical-restore"
	postgresRunnerDumpPath    = "/internal/postgres/logical-dump"
	postgresRunnerRestorePath = "/internal/postgres/logical-restore"
	mongoRunnerDumpPath       = "/internal/mongo/logical-dump"
	mongoRunnerRestorePath    = "/internal/mongo/logical-restore"
)

type backupRunnerClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewBackupRunnerClient(baseURL string, token string) appservices.BackupRunnerClient {
	return &backupRunnerClient{
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:      strings.TrimSpace(token),
		httpClient: &http.Client{},
	}
}

func (c *backupRunnerClient) RunMySQLLogicalDump(ctx context.Context, req *appservices.MySQLLogicalDumpRequest, writer io.Writer) (*appservices.BackupRunnerArtifact, error) {
	if req == nil {
		return nil, fmt.Errorf("mysql dump request is required")
	}
	payload, err := json.Marshal(map[string]any{
		"version":          req.Version,
		"host":             req.Host,
		"port":             req.Port,
		"username":         req.Username,
		"password":         req.Password,
		"database":         req.Database,
		"compression_type": req.CompressionType,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal mysql dump request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+mySQLRunnerDumpPath, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create mysql dump request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.setAuthHeader(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call backup runner dump: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.decodeRunnerError("mysql dump", resp)
	}
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return nil, fmt.Errorf("read mysql dump response: %w", err)
	}
	return &appservices.BackupRunnerArtifact{
		FileName: strings.TrimSpace(resp.Header.Get("X-Tango-Artifact-File-Name")),
		Metadata: map[string]any{
			"tool":             "mysqldump",
			"mysql_version":    strings.TrimSpace(resp.Header.Get("X-Tango-MySQL-Version")),
			"compression_type": strings.TrimSpace(resp.Header.Get("X-Tango-Compression-Type")),
		},
	}, nil
}

func (c *backupRunnerClient) RunMySQLLogicalRestore(ctx context.Context, req *appservices.MySQLLogicalRestoreRequest, reader io.Reader) error {
	if req == nil {
		return fmt.Errorf("mysql restore request is required")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fields := map[string]string{
		"version":          req.Version,
		"host":             req.Host,
		"port":             strconv.Itoa(req.Port),
		"username":         req.Username,
		"password":         req.Password,
		"database":         req.Database,
		"compression_type": string(req.CompressionType),
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return fmt.Errorf("write restore field %s: %w", key, err)
		}
	}
	part, err := writer.CreateFormFile("artifact", filepath.Base(req.Database)+restoreArtifactSuffix(req.CompressionType, "sql"))
	if err != nil {
		return fmt.Errorf("create restore artifact form file: %w", err)
	}
	if _, err := io.Copy(part, reader); err != nil {
		return fmt.Errorf("write restore artifact: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close restore multipart writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+mySQLRunnerRestorePath, &body)
	if err != nil {
		return fmt.Errorf("create mysql restore request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	c.setAuthHeader(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("call backup runner restore: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return c.decodeRunnerError("mysql restore", resp)
	}
	return nil
}

func (c *backupRunnerClient) RunPostgresLogicalDump(ctx context.Context, req *appservices.PostgresLogicalDumpRequest, writer io.Writer) (*appservices.BackupRunnerArtifact, error) {
	if req == nil {
		return nil, fmt.Errorf("postgres dump request is required")
	}
	payload, err := json.Marshal(map[string]any{
		"version":          req.Version,
		"host":             req.Host,
		"port":             req.Port,
		"username":         req.Username,
		"password":         req.Password,
		"database":         req.Database,
		"compression_type": req.CompressionType,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal postgres dump request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+postgresRunnerDumpPath, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create postgres dump request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.setAuthHeader(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call backup runner dump: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.decodeRunnerError("postgres dump", resp)
	}
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return nil, fmt.Errorf("read postgres dump response: %w", err)
	}
	return &appservices.BackupRunnerArtifact{
		FileName: strings.TrimSpace(resp.Header.Get("X-Tango-Artifact-File-Name")),
		Metadata: map[string]any{
			"tool":             "pg_dump",
			"postgres_version": strings.TrimSpace(resp.Header.Get("X-Tango-Postgres-Version")),
			"compression_type": strings.TrimSpace(resp.Header.Get("X-Tango-Compression-Type")),
			"dump_format":      strings.TrimSpace(resp.Header.Get("X-Tango-Postgres-Format")),
		},
	}, nil
}

func (c *backupRunnerClient) RunPostgresLogicalRestore(ctx context.Context, req *appservices.PostgresLogicalRestoreRequest, reader io.Reader) error {
	if req == nil {
		return fmt.Errorf("postgres restore request is required")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fields := map[string]string{
		"version":          req.Version,
		"host":             req.Host,
		"port":             strconv.Itoa(req.Port),
		"username":         req.Username,
		"password":         req.Password,
		"database":         req.Database,
		"compression_type": string(req.CompressionType),
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return fmt.Errorf("write restore field %s: %w", key, err)
		}
	}
	part, err := writer.CreateFormFile("artifact", filepath.Base(req.Database)+restoreArtifactSuffix(req.CompressionType, "dump"))
	if err != nil {
		return fmt.Errorf("create restore artifact form file: %w", err)
	}
	if _, err := io.Copy(part, reader); err != nil {
		return fmt.Errorf("write restore artifact: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close restore multipart writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+postgresRunnerRestorePath, &body)
	if err != nil {
		return fmt.Errorf("create postgres restore request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	c.setAuthHeader(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("call backup runner restore: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return c.decodeRunnerError("postgres restore", resp)
	}
	return nil
}

func (c *backupRunnerClient) RunMongoLogicalDump(ctx context.Context, req *appservices.MongoLogicalDumpRequest, writer io.Writer) (*appservices.BackupRunnerArtifact, error) {
	if req == nil {
		return nil, fmt.Errorf("mongo dump request is required")
	}
	payload, err := json.Marshal(map[string]any{
		"host":             req.Host,
		"port":             req.Port,
		"username":         req.Username,
		"password":         req.Password,
		"database":         req.Database,
		"auth_database":    req.AuthDatabase,
		"connection_uri":   req.ConnectionURI,
		"compression_type": req.CompressionType,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal mongo dump request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+mongoRunnerDumpPath, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create mongo dump request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.setAuthHeader(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call backup runner dump: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.decodeRunnerError("mongo dump", resp)
	}
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return nil, fmt.Errorf("read mongo dump response: %w", err)
	}
	return &appservices.BackupRunnerArtifact{
		FileName: strings.TrimSpace(resp.Header.Get("X-Tango-Artifact-File-Name")),
		Metadata: map[string]any{
			"tool":             "mongodump",
			"compression_type": strings.TrimSpace(resp.Header.Get("X-Tango-Compression-Type")),
			"database_name":    strings.TrimSpace(resp.Header.Get("X-Tango-Mongo-Database")),
		},
	}, nil
}

func (c *backupRunnerClient) RunMongoLogicalRestore(ctx context.Context, req *appservices.MongoLogicalRestoreRequest, reader io.Reader) error {
	if req == nil {
		return fmt.Errorf("mongo restore request is required")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fields := map[string]string{
		"host":             req.Host,
		"port":             strconv.Itoa(req.Port),
		"username":         req.Username,
		"password":         req.Password,
		"database":         req.Database,
		"auth_database":    req.AuthDatabase,
		"connection_uri":   req.ConnectionURI,
		"source_database":  req.SourceDatabase,
		"compression_type": string(req.CompressionType),
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return fmt.Errorf("write restore field %s: %w", key, err)
		}
	}
	part, err := writer.CreateFormFile("artifact", filepath.Base(req.Database)+restoreArtifactSuffix(req.CompressionType, "archive"))
	if err != nil {
		return fmt.Errorf("create restore artifact form file: %w", err)
	}
	if _, err := io.Copy(part, reader); err != nil {
		return fmt.Errorf("write restore artifact: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close restore multipart writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+mongoRunnerRestorePath, &body)
	if err != nil {
		return fmt.Errorf("create mongo restore request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	c.setAuthHeader(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("call backup runner restore: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return c.decodeRunnerError("mongo restore", resp)
	}
	return nil
}

func (c *backupRunnerClient) setAuthHeader(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func (c *backupRunnerClient) decodeRunnerError(operation string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = resp.Status
	}
	return fmt.Errorf("%s runner error: %s", operation, message)
}

func restoreArtifactSuffix(compressionType domain.BackupCompressionType, baseExtension string) string {
	if compressionType == domain.BackupCompressionGzip {
		return "." + baseExtension + ".gz"
	}
	return "." + baseExtension
}
