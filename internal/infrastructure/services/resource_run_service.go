package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"tango/internal/domain"
)

type ResourceRunService struct {
	resourceRepo domain.ResourceRepository
	runRepo      domain.ResourceRunRepository
	dockerRepo   domain.DockerRepository
	logger       *slog.Logger
	active       sync.Map // runID -> *LogBroadcaster
}

func NewResourceRunService(
	resourceRepo domain.ResourceRepository,
	runRepo domain.ResourceRunRepository,
	dockerRepo domain.DockerRepository,
	logger *slog.Logger,
) *ResourceRunService {
	return &ResourceRunService{
		resourceRepo: resourceRepo,
		runRepo:      runRepo,
		dockerRepo:   dockerRepo,
		logger:       logger,
	}
}

func (s *ResourceRunService) RunStartAsync(run *domain.ResourceRun) {
	b := newLogBroadcaster()
	s.active.Store(run.ID, b)

	go func() {
		if err := s.runStart(run, b); err != nil {
			s.logger.Error("resource start run failed", "run_id", run.ID, "resource_id", run.ResourceID, "err", err)
		}
	}()
}

// StartAfterBuild implements services.ResourceAutoStarter. It updates the resource
// image to the newly built tag, then immediately kicks off a start run.
func (s *ResourceRunService) StartAfterBuild(ctx context.Context, resourceID, imageTag string) error {
	// 1. Persist the new image on the resource (status → stopped)
	if err := s.resourceRepo.UpdateBuildComplete(ctx, resourceID, imageTag, ""); err != nil {
		return fmt.Errorf("update resource build complete: %w", err)
	}

	// 2. Create a new run record
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	runID := "resrun_" + hex.EncodeToString(b)

	run, err := domain.NewResourceRun(runID, resourceID)
	if err != nil {
		return err
	}
	saved, err := s.runRepo.Save(ctx, run)
	if err != nil {
		return fmt.Errorf("save resource run: %w", err)
	}

	// 3. Launch the start sequence asynchronously
	s.RunStartAsync(saved)
	return nil
}

func (s *ResourceRunService) Subscribe(runID string) (<-chan []byte, func()) {
	val, ok := s.active.Load(runID)
	if !ok {
		return nil, func() {}
	}
	return val.(*LogBroadcaster).Subscribe()
}

func (s *ResourceRunService) runStart(run *domain.ResourceRun, b *LogBroadcaster) error {
	ctx := context.Background()

	defer func() {
		b.closeAll()
		s.active.Delete(run.ID)
	}()

	logLine := func(line string) {
		fmt.Fprintln(b, line) //nolint:errcheck
	}

	fail := func(msg string, err error) error {
		errMsg := msg
		if err != nil {
			errMsg = msg + ": " + err.Error()
		}
		logLine("[ERROR] " + errMsg)
		now := time.Now().UTC()
		run.Status = domain.ResourceRunStatusFailed
		run.ErrorMsg = errMsg
		run.Logs = b.Snapshot()
		run.FinishedAt = &now
		if _, updateErr := s.runRepo.Update(ctx, run); updateErr != nil {
			s.logger.Error("update resource run on failure", "run_id", run.ID, "err", updateErr)
		}
		if resource, getErr := s.resourceRepo.GetByID(ctx, run.ResourceID); getErr == nil {
			_ = s.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusError, resource.ContainerID)
		}
		return fmt.Errorf("%s", errMsg)
	}

	updateStatus := func(status domain.ResourceRunStatus) {
		run.Status = status
		run.Logs = b.Snapshot()
		if _, err := s.runRepo.Update(ctx, run); err != nil {
			s.logger.Warn("update resource run status", "run_id", run.ID, "status", status, "err", err)
		}
	}

	now := time.Now().UTC()
	run.StartedAt = &now
	updateStatus(domain.ResourceRunStatusCheckingImage)

	resource, err := s.resourceRepo.GetByID(ctx, run.ResourceID)
	if err != nil {
		return fail("load resource", err)
	}

	imageRef := resource.Image
	if tag := strings.TrimSpace(resource.Tag); tag != "" {
		imageRef += ":" + tag
	}

	logLine("[check] loading resource metadata")
	logLine(fmt.Sprintf("[check] image reference: %s", imageRef))

	// Check for host-port conflicts with other running resources.
	for _, p := range resource.Ports {
		if p.HostPort <= 0 {
			continue
		}
		owner, err := s.resourceRepo.FindRunningByHostPort(ctx, p.HostPort)
		if err != nil {
			return fail("check port availability", err)
		}
		if owner != nil && owner.ID != resource.ID {
			return fail("port conflict", &domain.ErrHostPortConflict{
				Port:           p.HostPort,
				OccupiedByID:   owner.ID,
				OccupiedByName: owner.Name,
			})
		}
	}

	if s.dockerRepo == nil {
		return fail("docker is unavailable", nil)
	}

	images, err := s.dockerRepo.ListImages(ctx)
	if err != nil {
		return fail("list local images", err)
	}

	imageAvailable := false
	for _, img := range images {
		for _, tag := range img.Tags {
			if tag == imageRef {
				imageAvailable = true
				break
			}
		}
		if imageAvailable {
			break
		}
	}

	if !imageAvailable {
		updateStatus(domain.ResourceRunStatusPullingImage)
		logLine(fmt.Sprintf("[pull] pulling image %s", imageRef))
		if err := s.dockerRepo.PullImage(ctx, domain.PullImageInput{Reference: imageRef}); err != nil {
			return fail("pull image", err)
		}
		logLine(fmt.Sprintf("[pull] image ready: %s", imageRef))
	}

	containerID := resource.ContainerID
	if containerID == "" {
		updateStatus(domain.ResourceRunStatusCreating)
		logLine("[create] creating container")

		containerName, err := buildUniqueContainerName(ctx, s.dockerRepo, resource)
		if err != nil {
			return fail("allocate container name", err)
		}
		logLine(fmt.Sprintf("[create] using container name %s", containerName))

		ct, err := s.dockerRepo.CreateContainer(ctx, domain.CreateContainerInput{
			Name:         containerName,
			Image:        imageRef,
			Env:          buildResourceEnv(resource.EnvVars),
			PortBindings: buildResourcePortBindings(resource.Ports),
		})
		if err != nil {
			return fail("create container", err)
		}

		containerID = ct.ID
		logLine(fmt.Sprintf("[create] container created: %s", containerID))
	} else {
		logLine(fmt.Sprintf("[create] reusing container: %s", containerID))
	}

	updateStatus(domain.ResourceRunStatusStarting)
	logLine(fmt.Sprintf("[start] starting container %s", containerID))
	if err := s.dockerRepo.StartContainer(ctx, containerID); err != nil {
		return fail("start container", err)
	}

	if err := s.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusRunning, containerID); err != nil {
		return fail("update resource status", err)
	}

	logLine("[done] resource is running")
	done := time.Now().UTC()
	run.Status = domain.ResourceRunStatusDone
	run.FinishedAt = &done
	run.Logs = b.Snapshot()
	run.ErrorMsg = ""
	if _, err := s.runRepo.Update(ctx, run); err != nil {
		s.logger.Error("update resource run on success", "run_id", run.ID, "err", err)
		return err
	}
	return nil
}

func buildResourceEnv(items []domain.ResourceEnvVar) map[string]string {
	if len(items) == 0 {
		return nil
	}
	result := make(map[string]string, len(items))
	for _, item := range items {
		result[item.Key] = item.Value
	}
	return result
}

func buildResourcePortBindings(items []domain.ResourcePort) map[string]string {
	if len(items) == 0 {
		return nil
	}
	result := make(map[string]string, len(items))
	for _, item := range items {
		proto := item.Proto
		if strings.TrimSpace(proto) == "" {
			proto = "tcp"
		}
		result[fmt.Sprintf("%d/%s", item.InternalPort, proto)] = fmt.Sprintf("%d", item.HostPort)
	}
	return result
}
