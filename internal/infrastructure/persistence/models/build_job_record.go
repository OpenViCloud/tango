package models

import "time"

type BuildJobRecord struct {
	ID         string     `gorm:"primaryKey;type:varchar(64)"`
	Status     string     `gorm:"column:status;type:varchar(32);not null;index:idx_build_jobs_status"`
	GitURL     string     `gorm:"column:git_url;type:text;not null"`
	GitBranch  string     `gorm:"column:git_branch;type:varchar(255);not null"`
	ImageTag   string     `gorm:"column:image_tag;type:text;not null"`
	Logs       string     `gorm:"column:logs;type:text"`
	ErrorMsg   string     `gorm:"column:error_msg;type:text"`
	StartedAt  *time.Time `gorm:"column:started_at"`
	FinishedAt *time.Time `gorm:"column:finished_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;not null"`
}

func (BuildJobRecord) TableName() string {
	return "build_jobs"
}
