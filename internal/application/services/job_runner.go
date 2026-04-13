package services

import "context"

// JobRunner executes a one-shot job resource synchronously and waits for it to finish.
// Used by CreateResourceStackHandler to run init jobs (e.g. db migrate) before services.
type JobRunner interface {
	RunJobSync(ctx context.Context, resourceID string) error
}
