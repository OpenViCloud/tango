package services

import "context"

type ResourceRuntimeReconcileSummary struct {
	Checked           int `json:"checked"`
	Updated           int `json:"updated"`
	Running           int `json:"running"`
	Stopped           int `json:"stopped"`
	Errored           int `json:"errored"`
	MissingContainers int `json:"missing_containers"`
}

type ResourceRuntimeReconciler interface {
	ReconcileAll(ctx context.Context) (*ResourceRuntimeReconcileSummary, error)
	ReconcileResource(ctx context.Context, resourceID string) (*ResourceRuntimeReconcileSummary, error)
}
