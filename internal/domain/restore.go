package domain

import (
	"context"
	"strings"
	"time"
)

type RestoreStatus string

const (
	RestoreStatusPending    RestoreStatus = "pending"
	RestoreStatusInProgress RestoreStatus = "in_progress"
	RestoreStatusCompleted  RestoreStatus = "completed"
	RestoreStatusFailed     RestoreStatus = "failed"
	RestoreStatusCanceled   RestoreStatus = "canceled"
)

type Restore struct {
	ID                      string
	BackupID                string
	DatabaseSourceID        string
	Status                  RestoreStatus
	TargetHost              string
	TargetPort              int
	TargetUsername          string
	TargetPasswordEncrypted string
	TargetDatabaseName      string
	TargetAuthDatabase      string
	TargetURIEncrypted      string
	StartedAt               *time.Time
	CompletedAt             *time.Time
	DurationMs              int64
	FailMessage             string
	Metadata                map[string]any
	CreatedAt               time.Time
}

type CreateRestoreInput struct {
	ID                      string
	BackupID                string
	DatabaseSourceID        string
	Status                  RestoreStatus
	TargetHost              string
	TargetPort              int
	TargetUsername          string
	TargetPasswordEncrypted string
	TargetDatabaseName      string
	TargetAuthDatabase      string
	TargetURIEncrypted      string
	StartedAt               *time.Time
	CompletedAt             *time.Time
	DurationMs              int64
	FailMessage             string
	Metadata                map[string]any
}

type UpdateRestoreInput struct {
	Status                  RestoreStatus
	TargetHost              string
	TargetPort              int
	TargetUsername          string
	TargetPasswordEncrypted string
	TargetDatabaseName      string
	TargetAuthDatabase      string
	TargetURIEncrypted      string
	StartedAt               *time.Time
	CompletedAt             *time.Time
	DurationMs              int64
	FailMessage             string
	Metadata                map[string]any
}

func ValidateRestoreStatus(value string) (RestoreStatus, error) {
	switch RestoreStatus(strings.TrimSpace(value)) {
	case RestoreStatusPending, RestoreStatusInProgress, RestoreStatusCompleted, RestoreStatusFailed, RestoreStatusCanceled:
		return RestoreStatus(strings.TrimSpace(value)), nil
	default:
		return "", ErrInvalidInput
	}
}

type RestoreRepository interface {
	Create(ctx context.Context, input CreateRestoreInput) (*Restore, error)
	GetByID(ctx context.Context, id string) (*Restore, error)
	Update(ctx context.Context, id string, input UpdateRestoreInput) (*Restore, error)
}
