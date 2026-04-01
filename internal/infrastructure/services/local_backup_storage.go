package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type localBackupStorage struct{}

func NewLocalBackupStorage() appservices.StorageDriver {
	return &localBackupStorage{}
}

func (d *localBackupStorage) StoreFile(_ context.Context, storage *domain.Storage, key string, localPath string) (*appservices.StoredObject, error) {
	basePath, err := resolveLocalStorageBasePath(storage)
	if err != nil {
		return nil, err
	}
	destPath := filepath.Join(basePath, filepath.Clean(key))
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}
	src, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("open local artifact: %w", err)
	}
	defer src.Close()
	dst, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("create destination file: %w", err)
	}
	defer dst.Close()
	size, err := io.Copy(dst, src)
	if err != nil {
		_ = os.Remove(destPath)
		return nil, fmt.Errorf("copy artifact to local storage: %w", err)
	}
	return &appservices.StoredObject{Path: destPath, Size: size}, nil
}

func (d *localBackupStorage) LoadFile(_ context.Context, storage *domain.Storage, path string) (*appservices.LocalObject, error) {
	basePath, err := resolveLocalStorageBasePath(storage)
	if err != nil {
		return nil, err
	}
	cleanPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(basePath, cleanPath)
	}
	if _, err := os.Stat(cleanPath); err != nil {
		return nil, fmt.Errorf("open stored artifact: %w", err)
	}
	return &appservices.LocalObject{Path: cleanPath, Cleanup: func() error { return nil }}, nil
}

func resolveLocalStorageBasePath(storage *domain.Storage) (string, error) {
	if storage == nil {
		return "", fmt.Errorf("storage is required")
	}
	raw, _ := storage.Config["base_path"].(string)
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("local storage base_path is required")
	}
	return filepath.Clean(raw), nil
}
