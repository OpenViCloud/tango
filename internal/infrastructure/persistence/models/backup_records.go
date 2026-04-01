package models

import "time"

type DatabaseSourceRecord struct {
	ID                     string    `gorm:"primaryKey;type:varchar(64)"`
	Name                   string    `gorm:"column:name;type:varchar(255);not null"`
	DBType                 string    `gorm:"column:db_type;type:varchar(32);not null;index"`
	Host                   string    `gorm:"column:host;type:varchar(255);not null"`
	Port                   int       `gorm:"column:port;not null"`
	Username               string    `gorm:"column:username;type:varchar(255);not null"`
	PasswordEncrypted      string    `gorm:"column:password_encrypted;type:text"`
	DatabaseName           string    `gorm:"column:database_name;type:varchar(255);not null"`
	Version                string    `gorm:"column:version;type:varchar(64)"`
	IsTLSEnabled           bool      `gorm:"column:is_tls_enabled;not null;default:false"`
	AuthDatabase           string    `gorm:"column:auth_database;type:varchar(255)"`
	ConnectionURIEncrypted string    `gorm:"column:connection_uri_encrypted;type:text"`
	ResourceID             string    `gorm:"column:resource_id;type:varchar(64)"`
	CreatedAt              time.Time `gorm:"column:created_at;not null"`
	UpdatedAt              time.Time `gorm:"column:updated_at;not null"`
}

func (DatabaseSourceRecord) TableName() string { return "database_sources" }

type StorageRecord struct {
	ID                   string    `gorm:"primaryKey;type:varchar(64)"`
	Name                 string    `gorm:"column:name;type:varchar(255);not null"`
	Type                 string    `gorm:"column:type;type:varchar(32);not null"`
	ConfigJSON           string    `gorm:"column:config_json;type:text"`
	CredentialsEncrypted string    `gorm:"column:credentials_encrypted;type:text"`
	CreatedAt            time.Time `gorm:"column:created_at;not null"`
	UpdatedAt            time.Time `gorm:"column:updated_at;not null"`
}

func (StorageRecord) TableName() string { return "storages" }

type BackupConfigRecord struct {
	ID               string    `gorm:"primaryKey;type:varchar(64)"`
	DatabaseSourceID string    `gorm:"column:database_source_id;type:varchar(64);not null;uniqueIndex"`
	StorageID        string    `gorm:"column:storage_id;type:varchar(64);not null"`
	IsEnabled        bool      `gorm:"column:is_enabled;not null;default:true"`
	ScheduleType     string    `gorm:"column:schedule_type;type:varchar(32);not null"`
	TimeOfDay        string    `gorm:"column:time_of_day;type:varchar(16)"`
	IntervalHours    int       `gorm:"column:interval_hours"`
	RetentionType    string    `gorm:"column:retention_type;type:varchar(32);not null"`
	RetentionDays    int       `gorm:"column:retention_days"`
	RetentionCount   int       `gorm:"column:retention_count"`
	IsRetryIfFailed  bool      `gorm:"column:is_retry_if_failed;not null;default:false"`
	MaxRetryCount    int       `gorm:"column:max_retry_count"`
	EncryptionType   string    `gorm:"column:encryption_type;type:varchar(32);not null"`
	CompressionType  string    `gorm:"column:compression_type;type:varchar(32);not null"`
	BackupMethod     string    `gorm:"column:backup_method;type:varchar(32);not null"`
	CreatedAt        time.Time `gorm:"column:created_at;not null"`
	UpdatedAt        time.Time `gorm:"column:updated_at;not null"`
}

func (BackupConfigRecord) TableName() string { return "backup_configs" }

type BackupRecord struct {
	ID               string     `gorm:"primaryKey;type:varchar(64)"`
	DatabaseSourceID string     `gorm:"column:database_source_id;type:varchar(64);not null;index"`
	BackupConfigID   string     `gorm:"column:backup_config_id;type:varchar(64)"`
	StorageID        string     `gorm:"column:storage_id;type:varchar(64);not null"`
	Status           string     `gorm:"column:status;type:varchar(32);not null;index"`
	BackupMethod     string     `gorm:"column:backup_method;type:varchar(32);not null"`
	FileName         string     `gorm:"column:file_name;type:varchar(512)"`
	FilePath         string     `gorm:"column:file_path;type:text"`
	FileSizeBytes    int64      `gorm:"column:file_size_bytes"`
	ChecksumSHA256   string     `gorm:"column:checksum_sha256;type:varchar(128)"`
	StartedAt        *time.Time `gorm:"column:started_at"`
	CompletedAt      *time.Time `gorm:"column:completed_at"`
	DurationMs       int64      `gorm:"column:duration_ms"`
	FailMessage      string     `gorm:"column:fail_message;type:text"`
	EncryptionType   string     `gorm:"column:encryption_type;type:varchar(32);not null"`
	MetadataJSON     string     `gorm:"column:metadata_json;type:text"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null"`
}

func (BackupRecord) TableName() string { return "backups" }

type RestoreRecord struct {
	ID                      string     `gorm:"primaryKey;type:varchar(64)"`
	BackupID                string     `gorm:"column:backup_id;type:varchar(64);not null;index"`
	DatabaseSourceID        string     `gorm:"column:database_source_id;type:varchar(64)"`
	Status                  string     `gorm:"column:status;type:varchar(32);not null;index"`
	TargetHost              string     `gorm:"column:target_host;type:varchar(255)"`
	TargetPort              int        `gorm:"column:target_port"`
	TargetUsername          string     `gorm:"column:target_username;type:varchar(255)"`
	TargetPasswordEncrypted string     `gorm:"column:target_password_encrypted;type:text"`
	TargetDatabaseName      string     `gorm:"column:target_database_name;type:varchar(255)"`
	TargetAuthDatabase      string     `gorm:"column:target_auth_database;type:varchar(255)"`
	TargetURIEncrypted      string     `gorm:"column:target_uri_encrypted;type:text"`
	StartedAt               *time.Time `gorm:"column:started_at"`
	CompletedAt             *time.Time `gorm:"column:completed_at"`
	DurationMs              int64      `gorm:"column:duration_ms"`
	FailMessage             string     `gorm:"column:fail_message;type:text"`
	MetadataJSON            string     `gorm:"column:metadata_json;type:text"`
	CreatedAt               time.Time  `gorm:"column:created_at;not null"`
}

func (RestoreRecord) TableName() string { return "restores" }
