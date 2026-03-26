package domain

import "context"

type BuildJobRepository interface {
	Save(ctx context.Context, job *BuildJob) (*BuildJob, error)
	Update(ctx context.Context, job *BuildJob) (*BuildJob, error)
	GetByID(ctx context.Context, id string) (*BuildJob, error)
	List(ctx context.Context, opts BuildJobListOptions) (*BuildJobListResult, error)
}
