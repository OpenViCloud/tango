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

const (
	BuildJobSourceGit    = "git"
	BuildJobSourceUpload = "upload"
)

const (
	BuildJobModeAuto       = "auto"       // detect stack, generate Dockerfile if needed
	BuildJobModeDockerfile = "dockerfile" // use existing Dockerfile in the source
)

var (
	ErrBuildJobNotFound      = errors.New("build job not found")
	ErrBuildJobNotCancelable = errors.New("build job cannot be canceled in its current status")
)

type BuildJob struct {
	ID          string
	Status      BuildJobStatus
	SourceType  string // "git" | "upload"
	BuildMode   string // "auto" | "dockerfile"
	GitURL      string
	GitBranch   string
	ArchivePath string // temp path on disk, upload source only
	ArchiveName string // original filename, for display
	ImageTag    string
	ResourceID  string // optional: resource to auto-start after build
	Logs        string
	ErrorMsg    string
	StartedAt   *time.Time
	FinishedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewBuildJob(id, gitURL, gitBranch, buildMode, imageTag, resourceID string) (*BuildJob, error) {
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
	mode := strings.TrimSpace(buildMode)
	if mode == "" {
		mode = BuildJobModeAuto
	}
	return &BuildJob{
		ID:         id,
		Status:     BuildJobStatusQueued,
		SourceType: BuildJobSourceGit,
		BuildMode:  mode,
		GitURL:     strings.TrimSpace(gitURL),
		GitBranch:  branch,
		ImageTag:   strings.TrimSpace(imageTag),
		ResourceID: strings.TrimSpace(resourceID),
	}, nil
}

func NewBuildJobFromUpload(id, archivePath, archiveName, buildMode, imageTag, resourceID string) (*BuildJob, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("build job id is required")
	}
	if strings.TrimSpace(archivePath) == "" {
		return nil, errors.New("archive path is required")
	}
	if strings.TrimSpace(imageTag) == "" {
		return nil, errors.New("build job image_tag is required")
	}
	mode := strings.TrimSpace(buildMode)
	if mode == "" {
		mode = BuildJobModeAuto
	}
	return &BuildJob{
		ID:          id,
		Status:      BuildJobStatusQueued,
		SourceType:  BuildJobSourceUpload,
		BuildMode:   mode,
		ArchivePath: strings.TrimSpace(archivePath),
		ArchiveName: strings.TrimSpace(archiveName),
		ImageTag:    strings.TrimSpace(imageTag),
		ResourceID:  strings.TrimSpace(resourceID),
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
