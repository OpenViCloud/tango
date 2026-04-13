package command_test

import (
	"context"
	"errors"
	"testing"

	"tango/internal/application/command"
	"tango/internal/domain"
	infradb "tango/internal/infrastructure/db"
	"tango/internal/infrastructure/persistence/models"
	persistrepo "tango/internal/infrastructure/persistence/repositories"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeJobRunner struct {
	called []string // resource IDs passed to RunJobSync
	err    error    // error to return (nil = success)
}

func (f *fakeJobRunner) RunJobSync(_ context.Context, resourceID string) error {
	f.called = append(f.called, resourceID)
	return f.err
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func openStackTestDB(t *testing.T) *persistrepo.ResourceRepository {
	t.Helper()
	db, err := infradb.Open("sqlite", "file::memory:?cache=shared&mode=rwc")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = infradb.Close(db) })
	if err := infradb.Migrate(context.Background(), db, models.All()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return persistrepo.NewResourceRepository(db)
}

func newStackHandler(
	resourceRepo domain.ResourceRepository,
	templates []domain.ResourceStackTemplate,
	runner *fakeJobRunner,
) *command.CreateResourceStackHandler {
	createResource := command.NewCreateResourceHandler(resourceRepo, nil, nil, nil)
	return command.NewCreateResourceStackHandler(createResource, templates, runner)
}

// minimalTemplate returns a template with one service component and no volumes
// (volumes would require os.MkdirAll in tests).
func minimalTemplate(id string) domain.ResourceStackTemplate {
	return domain.ResourceStackTemplate{
		ID:    id,
		Image: "apache/airflow",
		Tags:  []string{"3.0.2"},
		SharedEnv: []domain.ResourceStackTemplateEnvVar{
			{Key: "SHARED_KEY", Value: "shared_val"},
		},
		Components: []domain.ResourceStackTemplateComponent{
			{
				ID:   "scheduler",
				Type: domain.ResourceTypeService,
				Cmd:  []string{"scheduler"},
			},
		},
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────────

// TestCreateResourceStack_CreatesServiceResources verifies that service components
// are persisted to the repository.
func TestCreateResourceStack_CreatesServiceResources(t *testing.T) {
	repo := openStackTestDB(t)
	runner := &fakeJobRunner{}
	h := newStackHandler(repo, []domain.ResourceStackTemplate{minimalTemplate("airflow")}, runner)

	result, err := h.Handle(context.Background(), command.CreateResourceStackCommand{
		TemplateID:    "airflow",
		NamePrefix:    "af",
		EnvironmentID: "env_1",
		CustomComponents: []command.CustomComponentInput{
			{ID: "scheduler", Type: domain.ResourceTypeService, Cmd: []string{"scheduler"}},
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(result.Resources) != 1 {
		t.Fatalf("len(Resources) = %d, want 1", len(result.Resources))
	}
	if result.Resources[0].Name != "af-scheduler" {
		t.Errorf("Name = %q, want %q", result.Resources[0].Name, "af-scheduler")
	}
	if result.Resources[0].Type != domain.ResourceTypeService {
		t.Errorf("Type = %q, want %q", result.Resources[0].Type, domain.ResourceTypeService)
	}
}

// TestCreateResourceStack_JobRunsBeforeServices verifies that job components are
// executed via JobRunner before service resources are created.
func TestCreateResourceStack_JobRunsBeforeServices(t *testing.T) {
	repo := openStackTestDB(t)
	runner := &fakeJobRunner{}
	h := newStackHandler(repo, []domain.ResourceStackTemplate{minimalTemplate("airflow")}, runner)

	result, err := h.Handle(context.Background(), command.CreateResourceStackCommand{
		TemplateID:    "airflow",
		NamePrefix:    "af",
		EnvironmentID: "env_1",
		CustomComponents: []command.CustomComponentInput{
			{ID: "db-migrate", Type: domain.ResourceTypeJob, Cmd: []string{"db", "migrate"}},
			{ID: "scheduler", Type: domain.ResourceTypeService, Cmd: []string{"scheduler"}},
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	// Job runner must have been called once (for db-migrate).
	if len(runner.called) != 1 {
		t.Fatalf("JobRunner called %d times, want 1", len(runner.called))
	}

	// Both resources are returned (job + service).
	if len(result.Resources) != 2 {
		t.Fatalf("len(Resources) = %d, want 2", len(result.Resources))
	}

	// Job resource comes first.
	if result.Resources[0].Name != "af-db-migrate" {
		t.Errorf("Resources[0].Name = %q, want %q", result.Resources[0].Name, "af-db-migrate")
	}
	if result.Resources[0].Type != domain.ResourceTypeJob {
		t.Errorf("Resources[0].Type = %q, want %q", result.Resources[0].Type, domain.ResourceTypeJob)
	}
}

// TestCreateResourceStack_JobFailurePreventsServices verifies that if a job fails,
// Handle() returns an error and no service resources are created.
func TestCreateResourceStack_JobFailurePreventsServices(t *testing.T) {
	repo := openStackTestDB(t)
	runner := &fakeJobRunner{err: errors.New("migration failed: exit code 1")}
	h := newStackHandler(repo, []domain.ResourceStackTemplate{minimalTemplate("airflow")}, runner)

	_, err := h.Handle(context.Background(), command.CreateResourceStackCommand{
		TemplateID:    "airflow",
		NamePrefix:    "af",
		EnvironmentID: "env_1",
		CustomComponents: []command.CustomComponentInput{
			{ID: "db-migrate", Type: domain.ResourceTypeJob, Cmd: []string{"db", "migrate"}},
			{ID: "scheduler", Type: domain.ResourceTypeService, Cmd: []string{"scheduler"}},
		},
	})
	if err == nil {
		t.Fatal("Handle() expected error, got nil")
	}

	// Scheduler resource must NOT have been created.
	resources, _ := repo.ListByEnvironment(context.Background(), "env_1")
	for _, r := range resources {
		if r.Name == "af-scheduler" {
			t.Errorf("scheduler resource was created despite job failure")
		}
	}
}

// TestCreateResourceStack_UnknownTemplate returns an error for unknown template IDs.
func TestCreateResourceStack_UnknownTemplate(t *testing.T) {
	repo := openStackTestDB(t)
	h := newStackHandler(repo, []domain.ResourceStackTemplate{minimalTemplate("airflow")}, nil)

	_, err := h.Handle(context.Background(), command.CreateResourceStackCommand{
		TemplateID:    "does-not-exist",
		EnvironmentID: "env_1",
		CustomComponents: []command.CustomComponentInput{
			{ID: "scheduler", Type: domain.ResourceTypeService},
		},
	})
	if err == nil {
		t.Fatal("Handle() expected error for unknown template, got nil")
	}
}

// TestCreateResourceStack_SharedEnvMergedWithComponentEnv verifies that shared env
// vars are applied to each component, and component env overrides shared env.
func TestCreateResourceStack_SharedEnvMergedWithComponentEnv(t *testing.T) {
	repo := openStackTestDB(t)
	runner := &fakeJobRunner{}

	tmpl := domain.ResourceStackTemplate{
		ID:    "airflow",
		Image: "apache/airflow",
		Tags:  []string{"3.0.2"},
		SharedEnv: []domain.ResourceStackTemplateEnvVar{
			{Key: "SHARED", Value: "from_shared"},
			{Key: "OVERRIDE", Value: "from_shared"},
		},
	}
	h := newStackHandler(repo, []domain.ResourceStackTemplate{tmpl}, runner)

	result, err := h.Handle(context.Background(), command.CreateResourceStackCommand{
		TemplateID:    "airflow",
		NamePrefix:    "af",
		EnvironmentID: "env_1",
		CustomComponents: []command.CustomComponentInput{
			{
				ID:   "scheduler",
				Type: domain.ResourceTypeService,
				Env: []command.ResourceEnvVarInput{
					{Key: "OVERRIDE", Value: "from_component"}, // should win
					{Key: "COMPONENT_ONLY", Value: "hello"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	envMap := make(map[string]string)
	for _, ev := range result.Resources[0].EnvVars {
		envMap[ev.Key] = ev.Value
	}

	if envMap["SHARED"] != "from_shared" {
		t.Errorf("SHARED = %q, want %q", envMap["SHARED"], "from_shared")
	}
	if envMap["OVERRIDE"] != "from_component" {
		t.Errorf("OVERRIDE = %q, want %q (component should win)", envMap["OVERRIDE"], "from_component")
	}
	if envMap["COMPONENT_ONLY"] != "hello" {
		t.Errorf("COMPONENT_ONLY = %q, want %q", envMap["COMPONENT_ONLY"], "hello")
	}
}

// TestCreateResourceStack_ImageTagFallback verifies that image and tag fall back
// to template defaults when the command leaves them empty.
func TestCreateResourceStack_ImageTagFallback(t *testing.T) {
	repo := openStackTestDB(t)
	h := newStackHandler(repo, []domain.ResourceStackTemplate{minimalTemplate("airflow")}, nil)

	result, err := h.Handle(context.Background(), command.CreateResourceStackCommand{
		TemplateID:    "airflow",
		NamePrefix:    "af",
		EnvironmentID: "env_1",
		Image:         "",   // empty → should use template default
		Tag:           "",   // empty → should use template default
		CustomComponents: []command.CustomComponentInput{
			{ID: "scheduler", Type: domain.ResourceTypeService},
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	r := result.Resources[0]
	if r.Image != "apache/airflow" {
		t.Errorf("Image = %q, want %q", r.Image, "apache/airflow")
	}
	if r.Tag != "3.0.2" {
		t.Errorf("Tag = %q, want %q", r.Tag, "3.0.2")
	}
}

// TestCreateResourceStack_NoComponents returns an error when the component list is empty.
func TestCreateResourceStack_NoComponents(t *testing.T) {
	repo := openStackTestDB(t)
	h := newStackHandler(repo, []domain.ResourceStackTemplate{minimalTemplate("airflow")}, nil)

	_, err := h.Handle(context.Background(), command.CreateResourceStackCommand{
		TemplateID:       "airflow",
		EnvironmentID:    "env_1",
		CustomComponents: []command.CustomComponentInput{},
	})
	if err == nil {
		t.Fatal("Handle() expected error for empty component list, got nil")
	}
}
