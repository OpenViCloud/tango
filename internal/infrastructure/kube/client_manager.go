package kube

import (
	"context"
	"fmt"
	"sync"

	"tango/internal/application/services"
	"tango/internal/domain"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeClientManager is a thread-safe, lazy-initialising pool of Kubernetes
// clients keyed by cluster ID. It implements domain.KubeClientFactory.
type KubeClientManager struct {
	clusterRepo domain.ClusterRepository
	cipher      services.SecretCipher

	mu      sync.RWMutex
	clients map[string]*kubeClientImpl // clusterID → cached client
}

// NewKubeClientManager creates a new KubeClientManager.
func NewKubeClientManager(repo domain.ClusterRepository, cipher services.SecretCipher) *KubeClientManager {
	return &KubeClientManager{
		clusterRepo: repo,
		cipher:      cipher,
		clients:     make(map[string]*kubeClientImpl),
	}
}

// GetClient returns a ready KubeClient for the given cluster ID.
// On the first call it fetches the encrypted kubeconfig from the DB, decrypts
// it, builds a *kubernetes.Clientset and caches the result.
func (m *KubeClientManager) GetClient(ctx context.Context, clusterID string) (domain.KubeClient, error) {
	// Fast path: check cache under read lock.
	m.mu.RLock()
	if c, ok := m.clients[clusterID]; ok {
		m.mu.RUnlock()
		return c, nil
	}
	m.mu.RUnlock()

	// Slow path: build and cache under write lock.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if c, ok := m.clients[clusterID]; ok {
		return c, nil
	}

	cluster, err := m.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("kube client manager: get cluster: %w", err)
	}
	if cluster.Kubeconfig == "" {
		return nil, fmt.Errorf("kube client manager: kubeconfig not available for cluster %s", clusterID)
	}

	raw, err := m.cipher.Decrypt(ctx, cluster.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("kube client manager: decrypt kubeconfig: %w", err)
	}

	restCfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(raw))
	if err != nil {
		return nil, fmt.Errorf("kube client manager: parse kubeconfig: %w", err)
	}

	cs, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("kube client manager: create clientset: %w", err)
	}

	impl := &kubeClientImpl{cs: cs}
	m.clients[clusterID] = impl
	return impl, nil
}

// InvalidateClient removes the cached client for the given cluster ID.
// Call this whenever the cluster's kubeconfig is rotated or updated.
func (m *KubeClientManager) InvalidateClient(clusterID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, clusterID)
}

// Compile-time assertion.
var _ domain.KubeClientFactory = (*KubeClientManager)(nil)
