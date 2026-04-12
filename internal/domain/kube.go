package domain

import "context"

// KubeNamespace is a lean domain representation of a Kubernetes namespace.
type KubeNamespace struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// KubePod is a lean domain representation of a Kubernetes pod.
type KubePod struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	NodeName  string `json:"node_name"`
	PodIP     string `json:"pod_ip"`
}

// KubeServicePort represents a port exposed by a Kubernetes service.
type KubeServicePort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port"`
	NodePort   int32  `json:"node_port,omitempty"`
	Protocol   string `json:"protocol"`
}

// KubeService is a lean domain representation of a Kubernetes service.
type KubeService struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	ClusterIP string            `json:"cluster_ip"`
	Ports     []KubeServicePort `json:"ports"`
}

// KubePersistentVolume is a lean domain representation of a Kubernetes PersistentVolume.
type KubePersistentVolume struct {
	Name             string `json:"name"`
	Capacity         string `json:"capacity"`
	AccessModes      string `json:"access_modes"`
	ReclaimPolicy    string `json:"reclaim_policy"`
	Status           string `json:"status"`
	StorageClassName string `json:"storage_class_name"`
}

// KubePersistentVolumeClaim is a lean domain representation of a Kubernetes PVC.
type KubePersistentVolumeClaim struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	Status           string `json:"status"`
	VolumeName       string `json:"volume_name"`
	Capacity         string `json:"capacity"`
	AccessModes      string `json:"access_modes"`
	StorageClassName string `json:"storage_class_name"`
}

// KubePodSpec describes the minimal spec needed to create a pod.
type KubePodSpec struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Image     string            `json:"image"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// KubeServiceSpec describes the minimal spec needed to create a service.
type KubeServiceSpec struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"` // ClusterIP | NodePort | LoadBalancer
	Selector  map[string]string `json:"selector"`
	Ports     []KubeServicePort `json:"ports"`
}

// KubeClient is the interface for interacting with a single Kubernetes cluster.
type KubeClient interface {
	ListNamespaces(ctx context.Context) ([]KubeNamespace, error)
	ListPods(ctx context.Context, namespace string) ([]KubePod, error)
	CreatePod(ctx context.Context, namespace string, spec KubePodSpec) (*KubePod, error)
	DeletePod(ctx context.Context, namespace, name string) error
	ListServices(ctx context.Context, namespace string) ([]KubeService, error)
	CreateService(ctx context.Context, namespace string, spec KubeServiceSpec) (*KubeService, error)
	DeleteService(ctx context.Context, namespace, name string) error
	ListPersistentVolumes(ctx context.Context) ([]KubePersistentVolume, error)
	ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]KubePersistentVolumeClaim, error)
}

// KubeClientFactory builds and caches KubeClient instances per cluster.
type KubeClientFactory interface {
	// GetClient returns a ready-to-use KubeClient for the given cluster ID.
	// It lazily initialises the client on first call and caches it.
	GetClient(ctx context.Context, clusterID string) (KubeClient, error)
	// InvalidateClient removes the cached client for a cluster.
	// Call this after the cluster's kubeconfig is rotated.
	InvalidateClient(clusterID string)
}
