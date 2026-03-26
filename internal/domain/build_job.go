package domain

import (
	"errors"
	"strings"
	"time"
)

type BuildJobStatus string

const (
	BuildJobStatusQueued     BuildJobStatus = "queued"
	BuildJobStatusCloning    BuildJobStatus = "cloning"
	BuildJobStatusDetecting  BuildJobStatus = "detecting"
	BuildJobStatusGenerating BuildJobStatus = "generating"
	BuildJobStatusBuilding   BuildJobStatus = "building"
	BuildJobStatusDone       BuildJobStatus = "done"
	BuildJobStatusFailed     BuildJobStatus = "failed"
	BuildJobStatusCanceled   BuildJobStatus = "canceled"
)

var (
	ErrBuildJobNotFound     = errors.New("build job not found")
	ErrBuildJobNotCancelable = errors.New("build job cannot be canceled in its current status")
)

type BuildJob struct {
	ID         string
	Status     BuildJobStatus
	GitURL     string
	GitBranch  string
	ImageTag   string
	Logs       string
	ErrorMsg   string
	StartedAt  *time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewBuildJob(id, gitURL, gitBranch, imageTag string) (*BuildJob, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("build job id is required")
	}
	if strings.TrimSpace(gitURL) == "" {
		return nil, errors.New("build job git_url is required")
	}
	if strings.TrimSpace(imageTag) == "" {
		return nil, errors.New("build job image_tag is required")
	}
	branch := strings.TrimSpace(gitBranch)
	if branch == "" {
		branch = "main"
	}
	return &BuildJob{
		ID:        id,
		Status:    BuildJobStatusQueued,
		GitURL:    strings.TrimSpace(gitURL),
		GitBranch: branch,
		ImageTag:  strings.TrimSpace(imageTag),
	}, nil
}

func (j *BuildJob) CanCancel() bool {
	switch j.Status {
	case BuildJobStatusQueued, BuildJobStatusCloning, BuildJobStatusDetecting,
		BuildJobStatusGenerating, BuildJobStatusBuilding:
		return true
	default:
		return false
	}
}

type BuildJobListOptions struct {
	PageIndex int
	PageSize  int
	Status    string
}

type BuildJobListResult struct {
	Items      []*BuildJob
	TotalItems int64
	PageIndex  int
	PageSize   int
}
