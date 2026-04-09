package ansible

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"text/template"

	"tango/internal/domain"
)

//go:embed playbooks/*.yml
var playbookFS embed.FS

// Phase represents a provisioning step.
type Phase struct {
	Name     string
	Playbook string // filename inside playbooks/
	ExtraVars map[string]string
}

// LogBroadcaster fans out log lines to multiple subscribers.
type LogBroadcaster struct {
	mu   sync.Mutex
	subs map[string][]chan []byte
}

func NewLogBroadcaster() *LogBroadcaster {
	return &LogBroadcaster{subs: make(map[string][]chan []byte)}
}

// Subscribe returns a channel that receives log lines for a cluster ID.
// The caller must call the returned unsubscribe func when done.
func (b *LogBroadcaster) Subscribe(clusterID string) (<-chan []byte, func()) {
	ch := make(chan []byte, 512)
	b.mu.Lock()
	b.subs[clusterID] = append(b.subs[clusterID], ch)
	b.mu.Unlock()

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		list := b.subs[clusterID]
		for i, c := range list {
			if c == ch {
				b.subs[clusterID] = append(list[:i], list[i+1:]...)
				close(ch)
				break
			}
		}
	}
	return ch, unsub
}

// publish sends a line to all subscribers of a cluster.
func (b *LogBroadcaster) publish(clusterID string, line []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs[clusterID] {
		select {
		case ch <- append([]byte(nil), line...):
		default: // drop if subscriber is slow
		}
	}
}

// closeAll closes all subscriber channels for a cluster (signals completion).
func (b *LogBroadcaster) closeAll(clusterID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs[clusterID] {
		close(ch)
	}
	delete(b.subs, clusterID)
}

// inventoryData holds template input for inventory.ini.
type inventoryData struct {
	Masters []nodeEntry
	Workers []nodeEntry
	SSHKey  string // path to private key file
}

type nodeEntry struct {
	Name   string
	IP     string // public IP (ansible_host)
	NodeIP string // private/internal IP (node_ip)
	User   string
	Port   int
}

const inventoryTpl = `[control_plane]
{{- range .Masters}}
{{.Name}} ansible_host={{.IP}} ansible_user={{.User}} node_ip={{.NodeIP}} ansible_port={{.Port}}
{{- end}}

[workers]
{{- range .Workers}}
{{.Name}} ansible_host={{.IP}} ansible_user={{.User}} node_ip={{.NodeIP}} ansible_port={{.Port}}
{{- end}}

[k8s:children]
control_plane
workers

[k8s:vars]
ansible_ssh_private_key_file={{.SSHKey}}
ansible_ssh_common_args='-o StrictHostKeyChecking=no'
`

// Runner executes Ansible playbooks for K8s cluster provisioning.
type Runner struct {
	broadcaster *LogBroadcaster
	sshManager  SSHKeyProvider
}

// SSHKeyProvider is the subset of ssh.Manager needed by Runner.
type SSHKeyProvider interface {
	PrivateKeyPEM(ctx context.Context) ([]byte, error)
}

func NewRunner(broadcaster *LogBroadcaster, sshManager SSHKeyProvider) *Runner {
	return &Runner{broadcaster: broadcaster, sshManager: sshManager}
}

// ProvisionCluster runs all three K8s provisioning phases asynchronously.
// Logs are broadcast to subscribers. clusterRepo is updated with final status.
func (r *Runner) ProvisionCluster(
	clusterID string,
	servers []*domain.Server,
	nodes []domain.ClusterNode,
	k8sVersion, podCIDR string,
	clusterRepo domain.ClusterRepository,
	onKubeconfig func(ctx context.Context, kubeconfigPath string),
) {
	go func() {
		ctx := context.Background()
		defer r.broadcaster.closeAll(clusterID)

		r.log(clusterID, "=== Tango Bootstrap Cluster ===")

		// Write private key to a temp file ansible can read
		privPEM, err := r.sshManager.PrivateKeyPEM(ctx)
		if err != nil {
			r.fail(ctx, clusterID, clusterRepo, fmt.Sprintf("get SSH key: %v", err))
			return
		}

		workDir, err := os.MkdirTemp("", "tango-cluster-"+clusterID+"-*")
		if err != nil {
			r.fail(ctx, clusterID, clusterRepo, fmt.Sprintf("create workdir: %v", err))
			return
		}
		defer os.RemoveAll(workDir)

		sshKeyPath := filepath.Join(workDir, "id_ed25519")
		if err := os.WriteFile(sshKeyPath, privPEM, 0600); err != nil {
			r.fail(ctx, clusterID, clusterRepo, fmt.Sprintf("write SSH key: %v", err))
			return
		}

		// Extract embedded playbooks to workDir
		if err := extractPlaybooks(workDir); err != nil {
			r.fail(ctx, clusterID, clusterRepo, fmt.Sprintf("extract playbooks: %v", err))
			return
		}

		// Build server lookup
		serverMap := make(map[string]*domain.Server, len(servers))
		for _, s := range servers {
			serverMap[s.ID] = s
		}

		// Split masters / workers
		var masters, workers []nodeEntry
		for _, n := range nodes {
			s, ok := serverMap[n.ServerID]
			if !ok {
				r.fail(ctx, clusterID, clusterRepo, fmt.Sprintf("server %s not found", n.ServerID))
				return
			}
			user := s.SSHUser
			if user == "" {
				user = "root"
			}
			port := s.SSHPort
			if port == 0 {
				port = 22
			}
			entry := nodeEntry{
				Name:   sanitizeName(s.Name),
				IP:     s.PublicIP,
				NodeIP: s.NodeIP(),
				User:   user,
				Port:   port,
			}
			if n.Role == domain.ClusterNodeRoleMaster {
				masters = append(masters, entry)
			} else {
				workers = append(workers, entry)
			}
		}

		// Generate inventory.ini
		invPath := filepath.Join(workDir, "inventory.ini")
		if err := renderInventory(inventoryData{
			Masters: masters,
			Workers: workers,
			SSHKey:  sshKeyPath,
		}, invPath); err != nil {
			r.fail(ctx, clusterID, clusterRepo, fmt.Sprintf("render inventory: %v", err))
			return
		}

		kubeconfigPath := filepath.Join(workDir, "kubeconfig.yaml")

		phases := []Phase{
			{
				Name:     "Installing prerequisites",
				Playbook: "k8s-install.yml",
				ExtraVars: map[string]string{
					"k8s_version": k8sVersion,
				},
			},
			{
				Name:     "Initializing control plane",
				Playbook: "master-init.yml",
				ExtraVars: map[string]string{
					"pod_cidr":        podCIDR,
					"kubeconfig_dest": kubeconfigPath,
				},
			},
			{
				Name:     "Joining workers",
				Playbook: "worker-join.yml",
			},
		}

		for _, phase := range phases {
			r.log(clusterID, fmt.Sprintf("\n>>> Phase: %s", phase.Name))
			if err := r.runPlaybook(clusterID, workDir, invPath, phase); err != nil {
				r.fail(ctx, clusterID, clusterRepo, fmt.Sprintf("phase %q failed: %v", phase.Name, err))
				return
			}
		}

		// Notify caller to fetch and store kubeconfig
		if onKubeconfig != nil {
			onKubeconfig(ctx, kubeconfigPath)
		}

		r.log(clusterID, "\n=== Bootstrap complete ===")
		if err := clusterRepo.UpdateStatus(ctx, clusterID, domain.ClusterStatusReady, ""); err != nil {
			r.log(clusterID, fmt.Sprintf("warning: update cluster status: %v", err))
		}
	}()
}

func (r *Runner) runPlaybook(clusterID, workDir, invPath string, phase Phase) error {
	args := []string{
		"-i", invPath,
		filepath.Join(workDir, phase.Playbook),
	}
	for k, v := range phase.ExtraVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "ANSIBLE_FORCE_COLOR=true")

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			r.log(clusterID, scanner.Text())
		}
	}()

	err := cmd.Run()
	pw.Close()
	wg.Wait()
	return err
}

func (r *Runner) log(clusterID, line string) {
	r.broadcaster.publish(clusterID, []byte(line))
}

func (r *Runner) fail(ctx context.Context, clusterID string, repo domain.ClusterRepository, msg string) {
	r.log(clusterID, "[ERROR] "+msg)
	_ = repo.UpdateStatus(ctx, clusterID, domain.ClusterStatusError, msg)
}

// renderInventory writes inventory.ini from template.
func renderInventory(data inventoryData, destPath string) error {
	tpl, err := template.New("inventory").Parse(inventoryTpl)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(destPath, buf.Bytes(), 0644)
}

// extractPlaybooks copies embedded playbook files to destDir.
func extractPlaybooks(destDir string) error {
	return fs.WalkDir(playbookFS, "playbooks", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := playbookFS.ReadFile(path)
		if err != nil {
			return err
		}
		dest := filepath.Join(destDir, filepath.Base(path))
		return os.WriteFile(dest, data, 0644)
	})
}

// sanitizeName converts a server name to a valid Ansible hostname.
func sanitizeName(name string) string {
	result := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		} else {
			result = append(result, '-')
		}
	}
	return string(result)
}

// PreviewInventory returns the inventory.ini content for a given cluster config (for UI preview).
func (r *Runner) PreviewInventory(servers []*domain.Server, nodes []domain.ClusterNode) (string, error) {
	return RenderInventoryPreview(servers, nodes)
}

// RenderInventoryPreview is the standalone version for use without a Runner.
func RenderInventoryPreview(servers []*domain.Server, nodes []domain.ClusterNode) (string, error) {
	serverMap := make(map[string]*domain.Server, len(servers))
	for _, s := range servers {
		serverMap[s.ID] = s
	}

	var masters, workers []nodeEntry
	for _, n := range nodes {
		s, ok := serverMap[n.ServerID]
		if !ok {
			continue
		}
		user := s.SSHUser
		if user == "" {
			user = "root"
		}
		port := s.SSHPort
		if port == 0 {
			port = 22
		}
		entry := nodeEntry{
			Name:   sanitizeName(s.Name),
			IP:     s.PublicIP,
			NodeIP: s.NodeIP(),
			User:   user,
			Port:   port,
		}
		if n.Role == domain.ClusterNodeRoleMaster {
			masters = append(masters, entry)
		} else {
			workers = append(workers, entry)
		}
	}

	tpl, err := template.New("inventory").Parse(inventoryTpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, inventoryData{
		Masters: masters,
		Workers: workers,
		SSHKey:  "<tango-managed-key>",
	}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

