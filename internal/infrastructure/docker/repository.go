package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"

	"tango/internal/domain"
)

// Repository wraps the Docker Engine API client.
type Repository struct {
	client *client.Client
}

// NewRepository creates a Docker client using environment variables
// (DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH, DOCKER_API_VERSION).
func NewRepository() (*Repository, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &Repository{client: cli}, nil
}

// Close releases the underlying connection.
func (r *Repository) Close() error {
	return r.client.Close()
}

// ListImages returns all local Docker images.
func (r *Repository) ListImages(ctx context.Context) ([]domain.Image, error) {
	items, err := r.client.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}

	result := make([]domain.Image, 0, len(items))
	for _, item := range items {
		digest := ""
		if len(item.RepoDigests) > 0 {
			digest = item.RepoDigests[0]
		}
		result = append(result, domain.Image{
			ID:      item.ID,
			Tags:    item.RepoTags,
			Size:    item.Size,
			Created: item.Created,
			Digest:  digest,
			InUse:   item.Containers,
		})
	}
	return result, nil
}

// PullImage pulls an image from a registry. It streams and discards output.
func (r *Repository) PullImage(ctx context.Context, input domain.PullImageInput) error {
	out, err := r.client.ImagePull(ctx, input.Reference, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", input.Reference, err)
	}
	defer out.Close()
	_, _ = io.Copy(io.Discard, out)
	return nil
}

// PullImageStream starts an image pull and returns the raw NDJSON event stream.
// The caller is responsible for closing the returned ReadCloser.
func (r *Repository) PullImageStream(ctx context.Context, reference string) (io.ReadCloser, error) {
	out, err := r.client.ImagePull(ctx, reference, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("pull image %s: %w", reference, err)
	}
	return out, nil
}

// RemoveImage removes an image by ID or tag.
func (r *Repository) RemoveImage(ctx context.Context, imageID string, force bool) error {
	_, err := r.client.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	if err != nil {
		return fmt.Errorf("remove image %s: %w", imageID, err)
	}
	return nil
}

// InspectContainer returns runtime details for the given container ID.
func (r *Repository) InspectContainer(ctx context.Context, containerID string) (domain.ContainerInfo, error) {
	info, err := r.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return domain.ContainerInfo{}, fmt.Errorf("inspect container %s: %w", containerID, err)
	}
	networks := map[string]string{}
	if info.NetworkSettings != nil {
		networks = make(map[string]string, len(info.NetworkSettings.Networks))
		for name, ep := range info.NetworkSettings.Networks {
			networks[name] = ep.IPAddress
		}
	}
	return domain.ContainerInfo{
		ID:       info.ID,
		Name:     strings.TrimPrefix(info.Name, "/"),
		Networks: networks,
	}, nil
}

func (r *Repository) GetContainerDetails(ctx context.Context, containerID string) (domain.ContainerDetails, error) {
	info, err := r.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return domain.ContainerDetails{}, classifyDockerError("inspect container", err)
	}

	networks := make(map[string]string, len(info.NetworkSettings.Networks))
	for name, ep := range info.NetworkSettings.Networks {
		networks[name] = ep.IPAddress
	}

	ports := make([]domain.ContainerPort, 0)
	if info.NetworkSettings != nil {
		for port, bindings := range info.NetworkSettings.Ports {
			privatePort, proto := splitNatPort(string(port))
			if len(bindings) == 0 {
				ports = append(ports, domain.ContainerPort{
					PrivatePort: privatePort,
					Type:        proto,
				})
				continue
			}
			for _, binding := range bindings {
				publicPort, _ := strconv.Atoi(binding.HostPort)
				ports = append(ports, domain.ContainerPort{
					IP:          binding.HostIP,
					PrivatePort: privatePort,
					PublicPort:  uint16(publicPort),
					Type:        proto,
				})
			}
		}
	}

	mounts := make([]domain.ContainerMount, 0, len(info.Mounts))
	for _, mountPoint := range info.Mounts {
		mounts = append(mounts, domain.ContainerMount{
			Type:        string(mountPoint.Type),
			Name:        mountPoint.Name,
			Source:      mountPoint.Source,
			Destination: mountPoint.Destination,
			Driver:      mountPoint.Driver,
			Mode:        mountPoint.Mode,
			RW:          mountPoint.RW,
		})
	}

	command := []string{}
	image := ""
	labels := map[string]string{}
	if info.Config != nil {
		if info.Path != "" {
			command = append(command, info.Path)
		}
		command = append(command, info.Args...)
		image = info.Config.Image
		labels = info.Config.Labels
	}

	state := ""
	status := ""
	startedAt := ""
	finishedAt := ""
	exitCode := 0
	stateError := ""
	if info.State != nil {
		state = string(info.State.Status)
		status = string(info.State.Status)
		startedAt = info.State.StartedAt
		finishedAt = info.State.FinishedAt
		exitCode = info.State.ExitCode
		stateError = info.State.Error
	}

	return domain.ContainerDetails{
		ID:           info.ID,
		Name:         strings.TrimPrefix(info.Name, "/"),
		Image:        image,
		ImageID:      info.Image,
		Command:      command,
		CreatedAt:    info.Created,
		State:        state,
		Status:       status,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		ExitCode:     exitCode,
		Error:        stateError,
		RestartCount: info.RestartCount,
		Ports:        ports,
		Labels:       labels,
		Networks:     networks,
		Mounts:       mounts,
	}, nil
}

func (r *Repository) GetContainerStats(ctx context.Context, containerID string) (domain.ContainerStats, error) {
	reader, err := r.client.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		return domain.ContainerStats{}, classifyDockerError("container stats", err)
	}
	defer reader.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(reader.Body).Decode(&stats); err != nil {
		return domain.ContainerStats{}, fmt.Errorf("decode container stats %s: %w", containerID, err)
	}

	rxBytes := uint64(0)
	txBytes := uint64(0)
	for _, networkStats := range stats.Networks {
		rxBytes += networkStats.RxBytes
		txBytes += networkStats.TxBytes
	}

	readBytes := uint64(0)
	writeBytes := uint64(0)
	for _, entry := range stats.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(entry.Op) {
		case "read":
			readBytes += entry.Value
		case "write":
			writeBytes += entry.Value
		}
	}

	memoryUsage := stats.MemoryStats.Usage
	inactiveFile, ok := stats.MemoryStats.Stats["inactive_file"]
	if ok && memoryUsage > inactiveFile {
		memoryUsage -= inactiveFile
	}

	memoryPercent := 0.0
	if stats.MemoryStats.Limit > 0 {
		memoryPercent = (float64(memoryUsage) / float64(stats.MemoryStats.Limit)) * 100
	}

	cpuPercent := calculateCPUPercent(stats)

	return domain.ContainerStats{
		ReadAt:           stats.Read.Format(time.RFC3339),
		CPUPercent:       cpuPercent,
		MemoryUsageBytes: memoryUsage,
		MemoryLimitBytes: stats.MemoryStats.Limit,
		MemoryPercent:    memoryPercent,
		NetworkRxBytes:   rxBytes,
		NetworkTxBytes:   txBytes,
		BlockReadBytes:   readBytes,
		BlockWriteBytes:  writeBytes,
		PidsCurrent:      stats.PidsStats.Current,
	}, nil
}

// ListContainers returns containers. Pass all=true to include stopped ones.
func (r *Repository) ListContainers(ctx context.Context, all bool) ([]domain.Container, error) {
	items, err := r.client.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	result := make([]domain.Container, 0, len(items))
	for _, item := range items {
		result = append(result, mapContainerSummary(item))
	}
	return result, nil
}

func splitNatPort(value string) (uint16, string) {
	port, proto, found := strings.Cut(value, "/")
	if !found {
		parsed, _ := strconv.Atoi(value)
		return uint16(parsed), "tcp"
	}
	parsed, _ := strconv.Atoi(port)
	return uint16(parsed), proto
}

func calculateCPUPercent(stats container.StatsResponse) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0
	}

	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0 {
		onlineCPUs = 1
	}

	return (cpuDelta / systemDelta) * onlineCPUs * 100
}

// EnsureNetwork makes sure a user-defined Docker network exists before
// containers are attached to it for Traefik/internal DNS resolution.
func (r *Repository) EnsureNetwork(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	if _, err := r.client.NetworkInspect(ctx, name, network.InspectOptions{}); err == nil {
		return nil
	}

	if _, err := r.client.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
	}); err != nil {
		if _, inspectErr := r.client.NetworkInspect(ctx, name, network.InspectOptions{}); inspectErr == nil {
			return nil
		}
		return fmt.Errorf("ensure network %s: %w", name, err)
	}

	return nil
}

// CreateContainer creates (but does not start) a new container.
func (r *Repository) CreateContainer(ctx context.Context, input domain.CreateContainerInput) (domain.Container, error) {
	portSet := nat.PortSet{}
	portMap := nat.PortMap{}

	for _, p := range input.ExposedPorts {
		port, err := parseContainerPort(p)
		if err != nil {
			return domain.Container{}, fmt.Errorf("parse exposed port %s: %w", p, err)
		}
		portSet[port] = struct{}{}
	}

	for containerPort, hostPort := range input.PortBindings {
		port, err := parseContainerPort(containerPort)
		if err != nil {
			return domain.Container{}, fmt.Errorf("parse port binding %s: %w", containerPort, err)
		}
		portSet[port] = struct{}{}
		portMap[port] = []nat.PortBinding{{HostPort: hostPort}}
	}

	cfg := &container.Config{
		Image:        input.Image,
		Cmd:          input.Cmd,
		Tty:          input.TTY,
		OpenStdin:    input.OpenStdin,
		Env:          envMapToSlice(input.Env),
		ExposedPorts: portSet,
		Labels:       input.Labels,
	}

	hostCfg := &container.HostConfig{
		AutoRemove:   input.AutoRemove,
		PortBindings: portMap,
		Binds:        input.Volumes,
		ExtraHosts:   []string{"host.docker.internal:host-gateway"},
		Resources: container.Resources{
			Memory:   input.MemoryLimit,
			NanoCPUs: input.CPULimit,
		},
	}

	// Apply restart policy unless AutoRemove is set (Docker disallows both).
	if !input.AutoRemove {
		hostCfg.RestartPolicy = container.RestartPolicy{Name: "unless-stopped"}
	}

	resp, err := r.client.ContainerCreate(ctx, cfg, hostCfg, nil, nil, input.Name)
	if err != nil {
		return domain.Container{}, classifyDockerError("create container", err)
	}

	// Join additional networks after creation (Docker only supports one network at create time)
	for _, netName := range input.Networks {
		if err := r.EnsureNetwork(ctx, netName); err != nil {
			return domain.Container{}, err
		}
		if err := r.client.NetworkConnect(ctx, netName, resp.ID, &network.EndpointSettings{}); err != nil {
			return domain.Container{}, fmt.Errorf("connect container %s to network %s: %w", resp.ID, netName, err)
		}
	}

	inspect, err := r.client.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return domain.Container{}, fmt.Errorf("inspect container after create: %w", err)
	}
	return mapInspect(inspect), nil
}

func (r *Repository) GetContainerLogs(ctx context.Context, containerID string, input domain.GetContainerLogsInput) ([]string, error) {
	reader, err := r.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Tail:       input.Tail,
	})
	if err != nil {
		return nil, fmt.Errorf("get container logs %s: %w", containerID, err)
	}
	defer reader.Close()

	var stdout strings.Builder
	var stderr strings.Builder
	if _, err := stdcopy.StdCopy(&stdout, &stderr, reader); err != nil {
		return nil, fmt.Errorf("decode container logs %s: %w", containerID, err)
	}

	combined := stdout.String()
	if stderr.Len() > 0 {
		if combined != "" && !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		combined += stderr.String()
	}

	lines := splitLogLines(combined)
	if len(lines) == 0 {
		return []string{}, nil
	}
	return lines, nil
}

// probeShell checks whether the given shell binary exists inside a container
// by running `test -x <shell>` as a detached exec and inspecting the exit code.
// Docker's ContainerExecCreate/Attach does NOT surface "file not found" errors
// until the stream is read, so we probe first to avoid returning a broken session.
func (r *Repository) probeShell(ctx context.Context, containerID, shell string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := r.client.ContainerExecCreate(probeCtx, containerID, container.ExecOptions{
		Cmd:          []string{"test", "-x", shell},
		AttachStdout: false,
		AttachStderr: false,
	})
	if err != nil {
		return false
	}
	if err := r.client.ContainerExecStart(probeCtx, resp.ID, container.ExecStartOptions{Detach: true}); err != nil {
		return false
	}
	// Poll until the test exec finishes (usually <50 ms)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		inspect, err := r.client.ContainerExecInspect(probeCtx, resp.ID)
		if err != nil {
			return false
		}
		if !inspect.Running {
			return inspect.ExitCode == 0
		}
		time.Sleep(30 * time.Millisecond)
	}
	return false
}

func (r *Repository) ExecContainer(ctx context.Context, containerID string, input domain.ContainerExecInput) (domain.ContainerExecSession, error) {
	shells := input.Shell
	if len(shells) == 0 {
		shells = []string{"/bin/bash", "/bin/sh"}
	}

	// Probe which shell actually exists before opening an interactive session.
	// Docker does not return an error from ContainerExecAttach when the binary
	// is missing — the failure only appears in the stream data, so the old
	// try/continue loop never triggered the fallback.
	chosen := ""
	for _, shell := range shells {
		if r.probeShell(ctx, containerID, shell) {
			chosen = shell
			break
		}
	}
	if chosen == "" {
		// Fallback: try /bin/sh unconditionally (always present on Linux)
		chosen = "/bin/sh"
	}

	execResp, err := r.client.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Cmd:          []string{chosen},
		Tty:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("exec container %s: %w", containerID, err)
	}

	attachResp, err := r.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		return nil, fmt.Errorf("exec attach %s: %w", containerID, err)
	}

	session := &execSession{
		client: r.client,
		execID: execResp.ID,
		conn:   attachResp.Conn,
		reader: attachResp.Reader,
	}

	if input.Cols > 0 && input.Rows > 0 {
		_ = session.Resize(ctx, input.Cols, input.Rows)
	}

	return session, nil
}

// StartContainer starts a stopped or newly created container.
func (r *Repository) StartContainer(ctx context.Context, containerID string) error {
	if err := r.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return classifyDockerError("start container", err)
	}
	return nil
}

// StopContainer sends a SIGTERM to the container and waits for it to exit.
func (r *Repository) StopContainer(ctx context.Context, containerID string) error {
	if err := r.client.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		return fmt.Errorf("stop container %s: %w", containerID, err)
	}
	return nil
}

// WaitContainer blocks until the container exits and returns its exit code.
func (r *Repository) WaitContainer(ctx context.Context, containerID string) (int64, error) {
	statusCh, errCh := r.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		return -1, fmt.Errorf("wait container %s: %w", containerID, err)
	case status := <-statusCh:
		return status.StatusCode, nil
	}
}

// RemoveContainer removes a container. Pass force=true to remove running containers.
func (r *Repository) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	f := filters.NewArgs(filters.Arg("id", containerID))
	_ = f
	if err := r.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: true,
	}); err != nil {
		return fmt.Errorf("remove container %s: %w", containerID, err)
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func mapContainerSummary(item dockertypes.Container) domain.Container {
	name := ""
	if len(item.Names) > 0 {
		name = strings.TrimPrefix(item.Names[0], "/")
	}

	ports := make([]domain.ContainerPort, 0, len(item.Ports))
	for _, p := range item.Ports {
		ports = append(ports, domain.ContainerPort{
			IP:          p.IP,
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Type:        p.Type,
		})
	}

	return domain.Container{
		ID:      item.ID,
		Name:    name,
		Image:   item.Image,
		ImageID: item.ImageID,
		State:   item.State,
		Status:  item.Status,
		Command: item.Command,
		Ports:   ports,
		Labels:  item.Labels,
	}
}

func mapInspect(item dockertypes.ContainerJSON) domain.Container {
	name := strings.TrimPrefix(item.Name, "/")

	ports := make([]domain.ContainerPort, 0)
	if item.NetworkSettings != nil {
		for port, bindings := range item.NetworkSettings.Ports {
			for _, b := range bindings {
				pub := uint16(0)
				fmt.Sscanf(b.HostPort, "%d", &pub)
				ports = append(ports, domain.ContainerPort{
					IP:          b.HostIP,
					PrivatePort: uint16(port.Int()),
					PublicPort:  pub,
					Type:        port.Proto(),
				})
			}
		}
	}

	state := ""
	status := ""
	if item.State != nil {
		state = item.State.Status
		if item.State.Running {
			status = "running"
		} else {
			status = "exited"
		}
	}

	cmd := ""
	if item.Config != nil {
		cmd = strings.Join(item.Config.Cmd, " ")
	}

	img := ""
	imgID := ""
	if item.Config != nil {
		img = item.Config.Image
	}
	imgID = item.Image

	return domain.Container{
		ID:      item.ID,
		Name:    name,
		Image:   img,
		ImageID: imgID,
		State:   state,
		Status:  status,
		Command: cmd,
		Ports:   ports,
		Labels:  item.Config.Labels,
	}
}

func envMapToSlice(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for k, v := range values {
		result = append(result, k+"="+v)
	}
	return result
}

func parseContainerPort(value string) (nat.Port, error) {
	portValue := value
	proto := "tcp"

	if strings.Contains(value, "/") {
		parts := strings.SplitN(value, "/", 2)
		portValue = parts[0]
		if len(parts) == 2 && parts[1] != "" {
			proto = parts[1]
		}
	}

	return nat.NewPort(proto, portValue)
}

func splitLogLines(raw string) []string {
	items := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
}

func tailValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "200"
	}
	if _, err := strconv.Atoi(value); err != nil {
		return "200"
	}
	return value
}

type execSession struct {
	client *client.Client
	execID string
	conn   net.Conn
	reader io.Reader
}

func (s *execSession) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *execSession) Write(p []byte) (int, error) {
	return s.conn.Write(p)
}

func (s *execSession) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

func (s *execSession) Resize(ctx context.Context, cols, rows uint) error {
	return s.client.ContainerExecResize(ctx, s.execID, container.ResizeOptions{
		Width:  cols,
		Height: rows,
	})
}

var _ domain.ContainerExecSession = (*execSession)(nil)

// classifyDockerError inspects the Docker daemon error message and converts
// known patterns into UserFacingError so the REST layer can return 400.
func classifyDockerError(op string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "port is already allocated"),
		strings.Contains(msg, "address already in use"),
		strings.Contains(msg, "Bind for") && strings.Contains(msg, "failed"):
		// Extract the port number if possible
		if idx := strings.Index(msg, "Bind for"); idx >= 0 {
			// "Bind for 0.0.0.0:5432 failed: port is already allocated"
			part := msg[idx:]
			fields := strings.Fields(part)
			if len(fields) >= 2 {
				addr := fields[2] // "0.0.0.0:5432"
				if _, port, e := net.SplitHostPort(addr); e == nil {
					return domain.NewUserFacingError(fmt.Sprintf("Port %s is already in use by another process or container", port))
				}
			}
		}
		return domain.NewUserFacingError("One or more ports are already in use by another process or container")
	case strings.Contains(msg, "already in use by container"):
		// "Conflict. The container name "/foo" is already in use by container ..."
		return domain.NewUserFacingError("A resource with this name already exists. Please choose a different name")
	case strings.Contains(msg, "No such image"):
		return domain.NewUserFacingError("Image not found. Check the image name and tag, then try again")
	case strings.Contains(msg, "pull access denied"),
		strings.Contains(msg, "unauthorized"):
		return domain.NewUserFacingError("Cannot pull image: access denied. The image may be private or the name may be incorrect")
	case strings.Contains(msg, "invalid reference format"):
		return domain.NewUserFacingError("Invalid image reference format. Please check the image name")
	default:
		return fmt.Errorf("%s: %w", op, err)
	}
}
