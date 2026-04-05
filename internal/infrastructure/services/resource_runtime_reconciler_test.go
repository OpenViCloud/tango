package services

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"tango/internal/domain"
	infradb "tango/internal/infrastructure/db"
	"tango/internal/infrastructure/persistence/models"
	persistrepo "tango/internal/infrastructure/persistence/repositories"

	"gorm.io/gorm"
)

type fakeDockerRepository struct {
	containers []domain.Container
	listErr    error
}

func (f *fakeDockerRepository) ListImages(context.Context) ([]domain.Image, error) {
	return nil, nil
}

func (f *fakeDockerRepository) PullImage(context.Context, domain.PullImageInput) error {
	return nil
}

func (f *fakeDockerRepository) RemoveImage(context.Context, string, bool) error {
	return nil
}

func (f *fakeDockerRepository) ListContainers(context.Context, bool) ([]domain.Container, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.containers, nil
}

func (f *fakeDockerRepository) GetContainerDetails(context.Context, string) (domain.ContainerDetails, error) {
	return domain.ContainerDetails{}, nil
}

func (f *fakeDockerRepository) GetContainerStats(context.Context, string) (domain.ContainerStats, error) {
	return domain.ContainerStats{}, nil
}

func (f *fakeDockerRepository) EnsureNetwork(context.Context, string) error {
	return nil
}

func (f *fakeDockerRepository) CreateContainer(context.Context, domain.CreateContainerInput) (domain.Container, error) {
	return domain.Container{}, nil
}

func (f *fakeDockerRepository) InspectContainer(context.Context, string) (domain.ContainerInfo, error) {
	return domain.ContainerInfo{}, nil
}

func (f *fakeDockerRepository) GetContainerLogs(context.Context, string, domain.GetContainerLogsInput) ([]string, error) {
	return nil, nil
}

func (f *fakeDockerRepository) ExecContainer(context.Context, string, domain.ContainerExecInput) (domain.ContainerExecSession, error) {
	return nil, nil
}

func (f *fakeDockerRepository) StartContainer(context.Context, string) error {
	return nil
}

func (f *fakeDockerRepository) StopContainer(context.Context, string) error {
	return nil
}

func (f *fakeDockerRepository) RemoveContainer(context.Context, string, bool) error {
	return nil
}

func (f *fakeDockerRepository) Close() error {
	return nil
}

type noopContainerExecSession struct{}

func (noopContainerExecSession) Read([]byte) (int, error)              { return 0, io.EOF }
func (noopContainerExecSession) Write(p []byte) (int, error)           { return len(p), nil }
func (noopContainerExecSession) Close() error                          { return nil }
func (noopContainerExecSession) Resize(context.Context, uint, uint) error {
	return nil
}

// fakeSwarmRepository is a test double for domain.SwarmRepository.
type fakeSwarmRepository struct {
	manager        bool
	serviceRunning map[string]bool
	serviceErr     error
}

func (f *fakeSwarmRepository) IsManager(context.Context) bool { return f.manager }

func (f *fakeSwarmRepository) CreateService(context.Context, domain.CreateServiceInput) (domain.SwarmService, error) {
	return domain.SwarmService{}, nil
}

func (f *fakeSwarmRepository) RemoveService(context.Context, string) error { return nil }

func (f *fakeSwarmRepository) ScaleService(_ context.Context, _ string, _ uint64) error { return nil }

func (f *fakeSwarmRepository) EnsureOverlayNetwork(context.Context, string) error { return nil }

func (f *fakeSwarmRepository) ListNodes(context.Context) ([]domain.SwarmNode, error) {
	return nil, nil
}

func (f *fakeSwarmRepository) ServiceRunning(_ context.Context, serviceID string) (bool, error) {
	if f.serviceErr != nil {
		return false, f.serviceErr
	}
	return f.serviceRunning[serviceID], nil
}

func (f *fakeSwarmRepository) GetServiceLogs(context.Context, string, string) ([]string, error) {
	return nil, nil
}

// ── Container-mode tests ──────────────────────────────────────────────────────

func TestResourceRuntimeReconcilerMarksMissingRunningContainerStopped(t *testing.T) {
	db := openResourceReconcileTestDB(t)
	resourceRepo := persistrepo.NewResourceRepository(db)
	ctx := context.Background()

	resource, err := resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            "res_1",
		Name:          "api",
		Type:          domain.ResourceTypeApp,
		Image:         "nginx",
		Tag:           "latest",
		EnvironmentID: "env_1",
		CreatedBy:     "user_1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusRunning, "ctr_missing"); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	reconciler := NewResourceRuntimeReconciler(
		resourceRepo,
		&fakeDockerRepository{containers: []domain.Container{}},
		nil, // no swarm
		slog.Default(),
	)

	summary, err := reconciler.ReconcileAll(ctx)
	if err != nil {
		t.Fatalf("ReconcileAll() error = %v", err)
	}
	if summary.Checked != 1 || summary.Updated != 1 || summary.MissingContainers != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	saved, err := resourceRepo.GetByID(ctx, resource.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Status != domain.ResourceStatusStopped {
		t.Fatalf("status = %s, want %s", saved.Status, domain.ResourceStatusStopped)
	}
	if saved.ContainerID != "" {
		t.Fatalf("container_id = %q, want empty", saved.ContainerID)
	}
}

func TestResourceRuntimeReconcilerMarksExistingContainerRunning(t *testing.T) {
	db := openResourceReconcileTestDB(t)
	resourceRepo := persistrepo.NewResourceRepository(db)
	ctx := context.Background()

	resource, err := resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            "res_2",
		Name:          "worker",
		Type:          domain.ResourceTypeService,
		Image:         "busybox",
		Tag:           "latest",
		EnvironmentID: "env_1",
		CreatedBy:     "user_1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusStopped, "ctr_running"); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	reconciler := NewResourceRuntimeReconciler(
		resourceRepo,
		&fakeDockerRepository{containers: []domain.Container{
			{ID: "ctr_running", State: "running"},
		}},
		nil, // no swarm
		slog.Default(),
	)

	summary, err := reconciler.ReconcileAll(ctx)
	if err != nil {
		t.Fatalf("ReconcileAll() error = %v", err)
	}
	if summary.Checked != 1 || summary.Updated != 1 || summary.Running != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	saved, err := resourceRepo.GetByID(ctx, resource.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Status != domain.ResourceStatusRunning {
		t.Fatalf("status = %s, want %s", saved.Status, domain.ResourceStatusRunning)
	}
	if saved.ContainerID != "ctr_running" {
		t.Fatalf("container_id = %q, want ctr_running", saved.ContainerID)
	}
}

// ── Swarm-mode tests ──────────────────────────────────────────────────────────

func TestResourceRuntimeReconcilerSwarmRunningService(t *testing.T) {
	db := openResourceReconcileTestDB(t)
	resourceRepo := persistrepo.NewResourceRepository(db)
	ctx := context.Background()

	resource, err := resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            "res_swarm_1",
		Name:          "svc-running",
		Type:          domain.ResourceTypeApp,
		Image:         "nginx",
		Tag:           "latest",
		EnvironmentID: "env_1",
		CreatedBy:     "user_1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	// Simulate resource in swarm mode: ContainerID holds a service ID.
	if err := resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusStopped, "svc_abc123"); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	swarm := &fakeSwarmRepository{
		manager:        true,
		serviceRunning: map[string]bool{"svc_abc123": true},
	}
	reconciler := NewResourceRuntimeReconciler(
		resourceRepo,
		&fakeDockerRepository{containers: []domain.Container{}}, // no containers
		swarm,
		slog.Default(),
	)

	summary, err := reconciler.ReconcileAll(ctx)
	if err != nil {
		t.Fatalf("ReconcileAll() error = %v", err)
	}
	if summary.Checked != 1 || summary.Updated != 1 || summary.Running != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	saved, err := resourceRepo.GetByID(ctx, resource.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Status != domain.ResourceStatusRunning {
		t.Fatalf("status = %s, want running", saved.Status)
	}
	if saved.ContainerID != "svc_abc123" {
		t.Fatalf("container_id = %q, want svc_abc123", saved.ContainerID)
	}
}

func TestResourceRuntimeReconcilerSwarmMissingServiceMarkedStopped(t *testing.T) {
	db := openResourceReconcileTestDB(t)
	resourceRepo := persistrepo.NewResourceRepository(db)
	ctx := context.Background()

	resource, err := resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            "res_swarm_2",
		Name:          "svc-gone",
		Type:          domain.ResourceTypeApp,
		Image:         "nginx",
		Tag:           "latest",
		EnvironmentID: "env_1",
		CreatedBy:     "user_1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusRunning, "svc_gone"); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	swarm := &fakeSwarmRepository{
		manager:        true,
		serviceRunning: map[string]bool{}, // service doesn't exist
	}
	reconciler := NewResourceRuntimeReconciler(
		resourceRepo,
		&fakeDockerRepository{containers: []domain.Container{}},
		swarm,
		slog.Default(),
	)

	summary, err := reconciler.ReconcileAll(ctx)
	if err != nil {
		t.Fatalf("ReconcileAll() error = %v", err)
	}
	if summary.Checked != 1 || summary.Updated != 1 || summary.MissingContainers != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	saved, err := resourceRepo.GetByID(ctx, resource.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Status != domain.ResourceStatusStopped {
		t.Fatalf("status = %s, want stopped", saved.Status)
	}
	if saved.ContainerID != "" {
		t.Fatalf("container_id = %q, want empty", saved.ContainerID)
	}
}

func TestResourceRuntimeReconcilerSwarmNotManagerFallsBackToContainers(t *testing.T) {
	db := openResourceReconcileTestDB(t)
	resourceRepo := persistrepo.NewResourceRepository(db)
	ctx := context.Background()

	resource, err := resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            "res_swarm_3",
		Name:          "fallback",
		Type:          domain.ResourceTypeApp,
		Image:         "nginx",
		Tag:           "latest",
		EnvironmentID: "env_1",
		CreatedBy:     "user_1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusStopped, "ctr_123"); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	// Swarm repo present but NOT a manager → container mode.
	swarmRepo := &fakeSwarmRepository{manager: false}
	reconciler := NewResourceRuntimeReconciler(
		resourceRepo,
		&fakeDockerRepository{containers: []domain.Container{
			{ID: "ctr_123", State: "running"},
		}},
		swarmRepo,
		slog.Default(),
	)

	summary, err := reconciler.ReconcileAll(ctx)
	if err != nil {
		t.Fatalf("ReconcileAll() error = %v", err)
	}
	if summary.Running != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	saved, err := resourceRepo.GetByID(ctx, resource.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Status != domain.ResourceStatusRunning {
		t.Fatalf("status = %s, want running", saved.Status)
	}
}

func openResourceReconcileTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := infradb.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = infradb.Close(db) })
	if err := infradb.Migrate(context.Background(), db, models.All()...); err != nil {
		t.Fatal(err)
	}
	return db
}
