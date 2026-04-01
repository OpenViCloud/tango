package domain

import (
	"context"
	"strings"
	"time"
)

type BackupScheduleType string
type BackupRetentionType string
type BackupEncryptionType string
type BackupCompressionType string
type BackupMethod string

const (
	BackupScheduleManualOnly BackupScheduleType = "manual_only"
	BackupScheduleHourly     BackupScheduleType = "hourly"
	BackupScheduleDaily      BackupScheduleType = "daily"
)

const (
	BackupRetentionNone  BackupRetentionType = "none"
	BackupRetentionDays  BackupRetentionType = "days"
	BackupRetentionCount BackupRetentionType = "count"
)

const (
	BackupEncryptionNone   BackupEncryptionType = "none"
	BackupEncryptionAES256 BackupEncryptionType = "aes256"
)

const (
	BackupCompressionNone BackupCompressionType = "none"
	BackupCompressionGzip BackupCompressionType = "gzip"
)

const (
	BackupMethodLogicalDump  BackupMethod = "logical_dump"
	BackupMethodPostgresPITR BackupMethod = "postgres_pitr"
)

type BackupConfig struct {
	ID               string
	DatabaseSourceID string
	StorageID        string
	IsEnabled        bool
	ScheduleType     BackupScheduleType
	TimeOfDay        string
	IntervalHours    int
	RetentionType    BackupRetentionType
	RetentionDays    int
	RetentionCount   int
	IsRetryIfFailed  bool
	MaxRetryCount    int
	EncryptionType   BackupEncryptionType
	CompressionType  BackupCompressionType
	BackupMethod     BackupMethod
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CreateBackupConfigInput struct {
	ID               string
	DatabaseSourceID string
	StorageID        string
	IsEnabled        bool
	ScheduleType     BackupScheduleType
	TimeOfDay        string
	IntervalHours    int
	RetentionType    BackupRetentionType
	RetentionDays    int
	RetentionCount   int
	IsRetryIfFailed  bool
	MaxRetryCount    int
	EncryptionType   BackupEncryptionType
	CompressionType  BackupCompressionType
	BackupMethod     BackupMethod
}

type UpdateBackupConfigInput struct {
	StorageID       string
	IsEnabled       bool
	ScheduleType    BackupScheduleType
	TimeOfDay       string
	IntervalHours   int
	RetentionType   BackupRetentionType
	RetentionDays   int
	RetentionCount  int
	IsRetryIfFailed bool
	MaxRetryCount   int
	EncryptionType  BackupEncryptionType
	CompressionType BackupCompressionType
	BackupMethod    BackupMethod
}

func ValidateBackupScheduleType(value string) (BackupScheduleType, error) {
	switch BackupScheduleType(strings.TrimSpace(value)) {
	case BackupScheduleManualOnly, BackupScheduleHourly, BackupScheduleDaily:
		return BackupScheduleType(strings.TrimSpace(value)), nil
	default:
		return "", ErrInvalidInput
	}
}

func ValidateBackupRetentionType(value string) (BackupRetentionType, error) {
	switch BackupRetentionType(strings.TrimSpace(value)) {
	case BackupRetentionNone, BackupRetentionDays, BackupRetentionCount:
		return BackupRetentionType(strings.TrimSpace(value)), nil
	default:
		return "", ErrInvalidInput
	}
}

func ValidateBackupEncryptionType(value string) (BackupEncryptionType, error) {
	switch BackupEncryptionType(strings.TrimSpace(value)) {
	case BackupEncryptionNone, BackupEncryptionAES256:
		return BackupEncryptionType(strings.TrimSpace(value)), nil
	default:
		return "", ErrInvalidInput
	}
}

func ValidateBackupCompressionType(value string) (BackupCompressionType, error) {
	switch BackupCompressionType(strings.TrimSpace(value)) {
	case BackupCompressionNone, BackupCompressionGzip:
		return BackupCompressionType(strings.TrimSpace(value)), nil
	default:
		return "", ErrInvalidInput
	}
}

func ValidateBackupMethod(value string) (BackupMethod, error) {
	switch BackupMethod(strings.TrimSpace(value)) {
	case BackupMethodLogicalDump, BackupMethodPostgresPITR:
		return BackupMethod(strings.TrimSpace(value)), nil
	default:
		return "", ErrInvalidInput
	}
}

type BackupConfigRepository interface {
	Create(ctx context.Context, input CreateBackupConfigInput) (*BackupConfig, error)
	GetByID(ctx context.Context, id string) (*BackupConfig, error)
	GetByDatabaseSourceID(ctx context.Context, databaseSourceID string) (*BackupConfig, error)
	Update(ctx context.Context, id string, input UpdateBackupConfigInput) (*BackupConfig, error)
}
