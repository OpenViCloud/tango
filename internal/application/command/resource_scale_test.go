package command_test

import (
	"context"
	"testing"

	"tango/internal/application/command"
	"tango/internal/domain"
	infradb "tango/internal/infrastructure/db"
	"tango/internal/infrastructure/persistence/models"
	persistrepo "tango/internal/infrastructure/persistence/repositories"

	"gorm.io/gorm"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeSwarmRepoScale struct {
	manager        bool
	scaledID       string
	scaledReplicas uint64
	scaleErr       error
}

func (f *fakeSwarmRepoScale) IsManager(_ context.Context) bool { return f.manager }
func (f *fakeSwarmRepoScale) CreateService(_ context.Context, _ domain.CreateServiceInput) (domain.SwarmService, error) {
	return domain.SwarmService{}, nil
}
func (f *fakeSwarmRepoScale) RemoveService(_ context.Context, _ string) error { return nil }
func (f *fakeSwarmRepoScale) ScaleService(_ context.Context, serviceID string, replicas uint64) error {
	if f.scaleErr != nil {
		return f.scaleErr
	}
	f.scaledID = serviceID
	f.scaledReplicas = replicas
	return nil
}
func (f *fakeSwarmRepoScale) EnsureOverlayNetwork(_ context.Context, _ string) error { return nil }
func (f *fakeSwarmRepoScale) ListNodes(_ context.Context) ([]domain.SwarmNode, error) {
	return nil, nil
}
func (f *fakeSwarmRepoScale) ServiceRunning(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (f *fakeSwarmRepoScale) GetServiceLogs(_ context.Context, _ string, _ string) ([]string, error) {
	return nil, nil
}

var _ domain.SwarmRepository = (*fakeSwarmRepoScale)(nil)

func openScaleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := infradb.Open("sqlite", "file::memory:?cache=shared&mode=rwc")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = infradb.Close(db) })
	if err := infradb.Migrate(context.Background(), db, models.All()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestScaleResourceHandler_PersistsReplicas verifies that ScaleResourceHandler
// saves the new replica count to the repository.
func TestScaleResourceHandler_PersistsReplicas(t *testing.T) {
	db := openScaleTestDB(t)
	ctx := context.Background()
	resourceRepo := persistrepo.NewResourceRepository(db)

	resource, err := resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            "res_scale_1",
		Name:          "my-app",
		Type:          domain.ResourceTypeApp,
		Image:         "nginx",
		EnvironmentID: "env_1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	handler := command.NewScaleResourceHandler(resourceRepo, nil) // no swarm
	if err := handler.Handle(ctx, command.ScaleResourceCommand{
		ID:       resource.ID,
		Replicas: 3,
	}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	saved, err := resourceRepo.GetByID(ctx, resource.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", saved.Replicas)
	}
}

// TestScaleResourceHandler_CallsSwarmScale verifies that ScaleResourceHandler
// calls SwarmRepository.ScaleService when in swarm mode and service is running.
func TestScaleResourceHandler_CallsSwarmScale(t *testing.T) {
	db := openScaleTestDB(t)
	ctx := context.Background()
	resourceRepo := persistrepo.NewResourceRepository(db)

	resource, err := resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            "res_scale_2",
		Name:          "my-svc",
		Type:          domain.ResourceTypeApp,
		Image:         "nginx",
		EnvironmentID: "env_1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	// Mark as running with a service ID.
	if err := resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusRunning, "svc_abc"); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	swarm := &fakeSwarmRepoScale{manager: true}
	handler := command.NewScaleResourceHandler(resourceRepo, swarm)
	if err := handler.Handle(ctx, command.ScaleResourceCommand{
		ID:       resource.ID,
		Replicas: 5,
	}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if swarm.scaledID != "svc_abc" {
		t.Errorf("scaledID = %q, want %q", swarm.scaledID, "svc_abc")
	}
	if swarm.scaledReplicas != 5 {
		t.Errorf("scaledReplicas = %d, want 5", swarm.scaledReplicas)
	}
}

// TestScaleResourceHandler_MinimumOne verifies that replicas < 1 are clamped to 1.
func TestScaleResourceHandler_MinimumOne(t *testing.T) {
	db := openScaleTestDB(t)
	ctx := context.Background()
	resourceRepo := persistrepo.NewResourceRepository(db)

	resource, err := resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            "res_scale_3",
		Name:          "clamp-test",
		Type:          domain.ResourceTypeApp,
		Image:         "nginx",
		EnvironmentID: "env_1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	handler := command.NewScaleResourceHandler(resourceRepo, nil)
	if err := handler.Handle(ctx, command.ScaleResourceCommand{
		ID:       resource.ID,
		Replicas: 0, // should be clamped to 1
	}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	saved, err := resourceRepo.GetByID(ctx, resource.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Replicas != 1 {
		t.Errorf("Replicas = %d, want 1 (clamped)", saved.Replicas)
	}
}

// TestScaleResourceHandler_NoSwarmSkipsScale verifies that when swarm is nil
// (single-node mode) no scale call is made but the DB is still updated.
func TestScaleResourceHandler_NoSwarmSkipsScale(t *testing.T) {
	db := openScaleTestDB(t)
	ctx := context.Background()
	resourceRepo := persistrepo.NewResourceRepository(db)

	resource, err := resourceRepo.Create(ctx, domain.CreateResourceInput{
		ID:            "res_scale_4",
		Name:          "no-swarm",
		Type:          domain.ResourceTypeApp,
		Image:         "nginx",
		EnvironmentID: "env_1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	_ = resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusRunning, "ctr_abc")

	handler := command.NewScaleResourceHandler(resourceRepo, nil) // swarm = nil
	if err := handler.Handle(ctx, command.ScaleResourceCommand{
		ID:       resource.ID,
		Replicas: 4,
	}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	saved, _ := resourceRepo.GetByID(ctx, resource.ID)
	if saved.Replicas != 4 {
		t.Errorf("Replicas = %d, want 4", saved.Replicas)
	}
}
