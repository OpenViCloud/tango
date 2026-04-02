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

func (noopContainerExecSession) Read([]byte) (int, error)  { return 0, io.EOF }
func (noopContainerExecSession) Write(p []byte) (int, error) { return len(p), nil }
func (noopContainerExecSession) Close() error              { return nil }
func (noopContainerExecSession) Resize(context.Context, uint, uint) error {
	return nil
}

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
