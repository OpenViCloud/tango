package domain

import (
	"context"
	"strings"
	"time"
)

type BackupStatus string

const (
	BackupStatusPending    BackupStatus = "pending"
	BackupStatusInProgress BackupStatus = "in_progress"
	BackupStatusCompleted  BackupStatus = "completed"
	BackupStatusFailed     BackupStatus = "failed"
	BackupStatusCanceled   BackupStatus = "canceled"
)

type Backup struct {
	ID               string
	DatabaseSourceID string
	BackupConfigID   string
	StorageID        string
	Status           BackupStatus
	BackupMethod     BackupMethod
	FileName         string
	FilePath         string
	FileSizeBytes    int64
	ChecksumSHA256   string
	StartedAt        *time.Time
	CompletedAt      *time.Time
	DurationMs       int64
	FailMessage      string
	EncryptionType   BackupEncryptionType
	Metadata         map[string]any
	CreatedAt        time.Time
}

type CreateBackupInput struct {
	ID               string
	DatabaseSourceID string
	BackupConfigID   string
	StorageID        string
	Status           BackupStatus
	BackupMethod     BackupMethod
	FileName         string
	FilePath         string
	FileSizeBytes    int64
	ChecksumSHA256   string
	StartedAt        *time.Time
	CompletedAt      *time.Time
	DurationMs       int64
	FailMessage      string
	EncryptionType   BackupEncryptionType
	Metadata         map[string]any
}

type UpdateBackupInput struct {
	Status         BackupStatus
	FileName       string
	FilePath       string
	FileSizeBytes  int64
	ChecksumSHA256 string
	StartedAt      *time.Time
	CompletedAt    *time.Time
	DurationMs     int64
	FailMessage    string
	Metadata       map[string]any
}

func ValidateBackupStatus(value string) (BackupStatus, error) {
	switch BackupStatus(strings.TrimSpace(value)) {
	case BackupStatusPending, BackupStatusInProgress, BackupStatusCompleted, BackupStatusFailed, BackupStatusCanceled:
		return BackupStatus(strings.TrimSpace(value)), nil
	default:
		return "", ErrInvalidInput
	}
}

type BackupRepository interface {
	Create(ctx context.Context, input CreateBackupInput) (*Backup, error)
	GetByID(ctx context.Context, id string) (*Backup, error)
	ListByDatabaseSourceID(ctx context.Context, databaseSourceID string) ([]*Backup, error)
	Update(ctx context.Context, id string, input UpdateBackupInput) (*Backup, error)
}
