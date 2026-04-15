package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"tango/internal/config"
	"tango/internal/domain"
)

type ResourceRunService struct {
	resourceRepo   domain.ResourceRepository
	runRepo        domain.ResourceRunRepository
	dockerRepo     domain.DockerRepository
	swarmRepo      domain.SwarmRepository
	domainRepo     domain.ResourceDomainRepository
	platformConfig domain.PlatformConfigRepository
	fileProvider   domain.TraefikFileProvider
	logger         *slog.Logger
	active         sync.Map // runID -> *LogBroadcaster
}

func NewResourceRunService(
	resourceRepo domain.ResourceRepository,
	runRepo domain.ResourceRunRepository,
	dockerRepo domain.DockerRepository,
	swarmRepo domain.SwarmRepository,
	domainRepo domain.ResourceDomainRepository,
	platformConfig domain.PlatformConfigRepository,
	fileProvider domain.TraefikFileProvider,
	logger *slog.Logger,
) *ResourceRunService {
	return &ResourceRunService{
		resourceRepo:   resourceRepo,
		runRepo:        runRepo,
		dockerRepo:     dockerRepo,
		swarmRepo:      swarmRepo,
		domainRepo:     domainRepo,
		platformConfig: platformConfig,
		fileProvider:   fileProvider,
		logger:         logger,
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

// RunJobSync implements application/services.JobRunner.
// It creates a run record, starts the job container, and blocks until it exits.
func (s *ResourceRunService) RunJobSync(ctx context.Context, resourceID string) error {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	runID := "resrun_" + hex.EncodeToString(b)

	run, err := domain.NewResourceRun(runID, resourceID)
	if err != nil {
		return fmt.Errorf("create job run record: %w", err)
	}
	saved, err := s.runRepo.Save(ctx, run)
	if err != nil {
		return fmt.Errorf("save job run record: %w", err)
	}

	broadcaster := newLogBroadcaster()
	s.active.Store(saved.ID, broadcaster)
	defer func() {
		broadcaster.closeAll()
		s.active.Delete(saved.ID)
	}()

	if err := s.runStart(saved, broadcaster); err != nil {
		return fmt.Errorf("job %s failed: %w", resourceID, err)
	}
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
	const defaultTraefikNetwork = "tango_net"

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

	if s.dockerRepo == nil {
		return fail("docker is unavailable", nil)
	}

	isSwarmMode := s.swarmRepo != nil && s.swarmRepo.IsManager(ctx)

	// Check for host-port conflicts only in single-node mode.
	// In swarm mode, services use overlay networking so no host ports are bound.
	if !isSwarmMode {
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

	// Resolve Traefik network and cert resolver from platform config once
	traefikNetwork := ""
	certResolver := ""
	if s.platformConfig != nil {
		if cfg, err := s.platformConfig.Get(ctx, domain.PlatformConfigTraefikNetwork); err == nil {
			traefikNetwork = cfg.Value
		}
		if cfg, err := s.platformConfig.Get(ctx, domain.PlatformConfigCertResolver); err == nil {
			certResolver = cfg.Value
		}
	}
	traefikNetwork = strings.TrimSpace(traefikNetwork)
	if traefikNetwork == "" {
		traefikNetwork = defaultTraefikNetwork
	}

	// ── Swarm mode ──────────────────────────────────────────────────────────────
	if isSwarmMode {
		return s.runStartSwarm(ctx, run, resource, imageRef, traefikNetwork, b, logLine, fail, updateStatus)
	}

	// ── Single-node container mode (default) ─────────────────────────────────
	// Determine the network(s) to join — always join tango_net so Traefik can reach
	// the container via Docker DNS (container name resolution).
	networks := []string{traefikNetwork}
	if err := s.dockerRepo.EnsureNetwork(ctx, traefikNetwork); err != nil {
		return fail("ensure shared docker network", err)
	}

	containerID := resource.ContainerID
	containerName := ""
	if containerID == "" {
		updateStatus(domain.ResourceRunStatusCreating)
		logLine("[create] creating container")

		containerName, err = buildUniqueContainerName(ctx, s.dockerRepo, resource)
		if err != nil {
			return fail("allocate container name", err)
		}
		logLine(fmt.Sprintf("[create] using container name %s", containerName))

		mountRoot := config.DefaultResourceMountRootHost
		if s.platformConfig != nil {
			if cfg, err := s.platformConfig.Get(ctx, domain.PlatformConfigResourceMountRoot); err == nil {
				if value := strings.TrimSpace(cfg.Value); value != "" {
					mountRoot = value
				}
			}
		}
		mounts, err := domain.ResolveResourceMounts(resource.Config, mountRoot)
		if err != nil {
			return fail("validate resource volumes", err)
		}
		for _, hostPath := range mounts.HostPaths {
			if err := os.MkdirAll(hostPath, 0o755); err != nil {
				return fail(fmt.Sprintf("prepare resource volume %s", hostPath), err)
			}
		}
		if err := domain.WriteVolumeFiles(resource.Config, mountRoot, buildResourceEnv(resource.EnvVars)); err != nil {
			return fail("write volume config files", err)
		}

		ct, err := s.dockerRepo.CreateContainer(ctx, domain.CreateContainerInput{
			Name:         containerName,
			Image:        imageRef,
			Env:          buildResourceEnv(resource.EnvVars),
			PortBindings: buildResourcePortBindings(resource.Ports),
			Volumes:      mounts.Binds,
			Cmd:          buildResourceCmd(resource.Config),
			Networks:     networks,
			MemoryLimit:  resource.MemoryLimit,
			CPULimit:     resource.CPULimit,
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

	// ── Job mode: wait for container to exit ────────────────────────────────
	if resource.Type == domain.ResourceTypeJob {
		logLine("[job] waiting for job container to finish…")
		exitCode, err := s.dockerRepo.WaitContainer(ctx, containerID)
		if err != nil {
			return fail("wait job container", err)
		}
		logLine(fmt.Sprintf("[job] container exited with code %d", exitCode))
		if exitCode != 0 {
			return fail(fmt.Sprintf("job failed with exit code %d", exitCode), nil)
		}
		if err := s.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusCompleted, containerID); err != nil {
			return fail("update job status", err)
		}
		logLine("[done] job completed successfully")
		done := time.Now().UTC()
		run.Status = domain.ResourceRunStatusDone
		run.FinishedAt = &done
		run.Logs = b.Snapshot()
		run.ErrorMsg = ""
		if _, err := s.runRepo.Update(ctx, run); err != nil {
			s.logger.Error("update resource run on success (job)", "run_id", run.ID, "err", err)
			return err
		}
		return nil
	}

	// Resolve container name if reusing an existing container
	if containerName == "" && s.dockerRepo != nil {
		if info, inspectErr := s.dockerRepo.InspectContainer(ctx, containerID); inspectErr == nil {
			containerName = info.Name
		} else {
			s.logger.Warn("traefik config: inspect container failed when resolving name",
				"resource_id", resource.ID, "container_id", containerID, "err", inspectErr)
		}
	}

	// Write Traefik file config after container is running
	if s.fileProvider != nil && containerName != "" && s.domainRepo != nil {
		domains, domainErr := s.domainRepo.ListByResource(ctx, resource.ID)
		if domainErr != nil {
			s.logger.Warn("traefik config: list domains failed", "resource_id", resource.ID, "err", domainErr)
		} else if len(domains) == 0 {
			s.logger.Debug("traefik config: no domains configured for resource, skipping config write",
				"resource_id", resource.ID)
		} else {
			if err := s.fileProvider.Write(resource.ID, domains, containerName, certResolver); err != nil {
				s.logger.Warn("traefik config: write failed",
					"resource_id", resource.ID, "container", containerName, "err", err)
			} else {
				logLine("[traefik] routing config written")
				s.logger.Debug("traefik config: written",
					"resource_id", resource.ID, "container", containerName, "domains", len(domains))
			}
		}
	} else if s.fileProvider != nil && containerName == "" {
		s.logger.Warn("traefik config: skipping write — could not resolve container name",
			"resource_id", resource.ID, "container_id", containerID)
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

// runStartSwarm handles resource start in Docker Swarm mode.
// It creates a swarm service instead of a plain container; Traefik reaches it
// via the overlay network DNS using the service name.
func (s *ResourceRunService) runStartSwarm(
	ctx context.Context,
	run *domain.ResourceRun,
	resource *domain.Resource,
	imageRef string,
	overlayNetwork string,
	b *LogBroadcaster,
	logLine func(string),
	fail func(string, error) error,
	updateStatus func(domain.ResourceRunStatus),
) error {
	serviceID := resource.ContainerID // reuse field; stores swarm service ID in cluster mode
	serviceName := normalizeContainerName(resource.Name)
	if serviceName == "" {
		serviceName = fmt.Sprintf("resource-%s", shortResourceID(resource.ID))
	}

	if serviceID == "" {
		updateStatus(domain.ResourceRunStatusCreating)
		logLine("[swarm] creating service")

		mountRoot := config.DefaultResourceMountRootHost
		if s.platformConfig != nil {
			if cfg, err := s.platformConfig.Get(ctx, domain.PlatformConfigResourceMountRoot); err == nil {
				if value := strings.TrimSpace(cfg.Value); value != "" {
					mountRoot = value
				}
			}
		}
		mounts, err := domain.ResolveResourceMounts(resource.Config, mountRoot)
		if err != nil {
			return fail("validate resource volumes", err)
		}
		for _, hostPath := range mounts.HostPaths {
			if err := os.MkdirAll(hostPath, 0o755); err != nil {
				return fail(fmt.Sprintf("prepare resource volume %s", hostPath), err)
			}
		}
		if err := domain.WriteVolumeFiles(resource.Config, mountRoot, buildResourceEnv(resource.EnvVars)); err != nil {
			return fail("write volume config files", err)
		}

		nodeID := ""
		if resource.NodeID != nil {
			nodeID = *resource.NodeID
		}

		replicas := uint64(resource.Replicas)
		if replicas == 0 {
			replicas = 1
		}
		svc, err := s.swarmRepo.CreateService(ctx, domain.CreateServiceInput{
			Name:        serviceName,
			Image:       imageRef,
			Cmd:         buildResourceCmd(resource.Config),
			Env:         buildResourceEnv(resource.EnvVars),
			Volumes:     mounts.Binds,
			Networks:    []string{overlayNetwork},
			NodeID:      nodeID,
			Replicas:    replicas,
			MemoryLimit: resource.MemoryLimit,
			CPULimit:    resource.CPULimit,
		})
		if err != nil {
			return fail("create swarm service", err)
		}
		serviceID = svc.ID
		logLine(fmt.Sprintf("[swarm] service created: %s (id=%s)", serviceName, serviceID))
	} else {
		logLine(fmt.Sprintf("[swarm] reusing service: %s", serviceID))
	}

	updateStatus(domain.ResourceRunStatusStarting)

	// Write Traefik routing — service name resolves via overlay DNS.
	certResolver := ""
	if s.platformConfig != nil {
		if cfg, err := s.platformConfig.Get(ctx, domain.PlatformConfigCertResolver); err == nil {
			certResolver = cfg.Value
		}
	}
	if s.fileProvider != nil && s.domainRepo != nil {
		if domains, err := s.domainRepo.ListByResource(ctx, resource.ID); err == nil && len(domains) > 0 {
			if err := s.fileProvider.Write(resource.ID, domains, serviceName, certResolver); err != nil {
				s.logger.Warn("write traefik file config (swarm)", "resource_id", resource.ID, "err", err)
			} else {
				logLine("[traefik] routing config written")
			}
		}
	}

	if err := s.resourceRepo.UpdateStatus(ctx, resource.ID, domain.ResourceStatusRunning, serviceID); err != nil {
		return fail("update resource status", err)
	}

	logLine("[done] resource is running (swarm)")
	done := time.Now().UTC()
	run.Status = domain.ResourceRunStatusDone
	run.FinishedAt = &done
	run.Logs = b.Snapshot()
	run.ErrorMsg = ""
	if _, err := s.runRepo.Update(ctx, run); err != nil {
		s.logger.Error("update resource run on success (swarm)", "run_id", run.ID, "err", err)
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

func buildResourceCmd(cfg map[string]any) []string {
	if cfg == nil {
		return nil
	}
	raw, ok := cfg["cmd"]
	if !ok {
		return nil
	}
	switch items := raw.(type) {
	case []string:
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, item)
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
