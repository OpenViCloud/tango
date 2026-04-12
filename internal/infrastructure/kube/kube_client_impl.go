package kube

import (
	"context"
	"fmt"
	"strconv"

	"tango/internal/domain"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// kubeClientImpl implements domain.KubeClient using a real *kubernetes.Clientset.
type kubeClientImpl struct {
	cs *kubernetes.Clientset
}

// ListNamespaces lists all namespaces in the cluster.
func (c *kubeClientImpl) ListNamespaces(ctx context.Context) ([]domain.KubeNamespace, error) {
	list, err := c.cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}
	result := make([]domain.KubeNamespace, 0, len(list.Items))
	for _, ns := range list.Items {
		result = append(result, domain.KubeNamespace{
			Name:   ns.Name,
			Status: string(ns.Status.Phase),
		})
	}
	return result, nil
}

// ListPods lists all pods in the given namespace ("" = all namespaces).
func (c *kubeClientImpl) ListPods(ctx context.Context, namespace string) ([]domain.KubePod, error) {
	list, err := c.cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}
	result := make([]domain.KubePod, 0, len(list.Items))
	for _, p := range list.Items {
		result = append(result, domain.KubePod{
			Name:      p.Name,
			Namespace: p.Namespace,
			Status:    string(p.Status.Phase),
			NodeName:  p.Spec.NodeName,
			PodIP:     p.Status.PodIP,
		})
	}
	return result, nil
}

// CreatePod creates a pod with a single container using the provided spec.
func (c *kubeClientImpl) CreatePod(ctx context.Context, namespace string, spec domain.KubePodSpec) (*domain.KubePod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: namespace,
			Labels:    spec.Labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  spec.Name,
					Image: spec.Image,
				},
			},
		},
	}
	created, err := c.cs.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create pod: %w", err)
	}
	return &domain.KubePod{
		Name:      created.Name,
		Namespace: created.Namespace,
		Status:    string(created.Status.Phase),
		NodeName:  created.Spec.NodeName,
		PodIP:     created.Status.PodIP,
	}, nil
}

// DeletePod deletes the named pod from the given namespace.
func (c *kubeClientImpl) DeletePod(ctx context.Context, namespace, name string) error {
	if err := c.cs.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("delete pod: %w", err)
	}
	return nil
}

// ListServices lists all services in the given namespace ("" = all namespaces).
func (c *kubeClientImpl) ListServices(ctx context.Context, namespace string) ([]domain.KubeService, error) {
	list, err := c.cs.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	result := make([]domain.KubeService, 0, len(list.Items))
	for _, svc := range list.Items {
		result = append(result, toKubeService(svc))
	}
	return result, nil
}

// CreateService creates a Kubernetes service from the provided spec.
func (c *kubeClientImpl) CreateService(ctx context.Context, namespace string, spec domain.KubeServiceSpec) (*domain.KubeService, error) {
	ports := make([]corev1.ServicePort, 0, len(spec.Ports))
	for _, p := range spec.Ports {
		sp := corev1.ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: parseTargetPort(p.TargetPort),
			Protocol:   corev1.Protocol(p.Protocol),
		}
		if p.NodePort != 0 {
			sp.NodePort = p.NodePort
		}
		ports = append(ports, sp)
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(spec.Type),
			Selector: spec.Selector,
			Ports:    ports,
		},
	}
	created, err := c.cs.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create service: %w", err)
	}
	result := toKubeService(*created)
	return &result, nil
}

// DeleteService deletes the named service from the given namespace.
func (c *kubeClientImpl) DeleteService(ctx context.Context, namespace, name string) error {
	if err := c.cs.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	return nil
}

// ListPersistentVolumes lists all PersistentVolumes in the cluster.
func (c *kubeClientImpl) ListPersistentVolumes(ctx context.Context) ([]domain.KubePersistentVolume, error) {
	list, err := c.cs.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list persistent volumes: %w", err)
	}
	result := make([]domain.KubePersistentVolume, 0, len(list.Items))
	for _, pv := range list.Items {
		capacity := ""
		if q, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
			capacity = q.String()
		}
		accessModes := accessModesToString(pv.Spec.AccessModes)
		result = append(result, domain.KubePersistentVolume{
			Name:             pv.Name,
			Capacity:         capacity,
			AccessModes:      accessModes,
			ReclaimPolicy:    string(pv.Spec.PersistentVolumeReclaimPolicy),
			Status:           string(pv.Status.Phase),
			StorageClassName: pv.Spec.StorageClassName,
		})
	}
	return result, nil
}

// ListPersistentVolumeClaims lists all PVCs in the given namespace ("" = all namespaces).
func (c *kubeClientImpl) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]domain.KubePersistentVolumeClaim, error) {
	list, err := c.cs.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list persistent volume claims: %w", err)
	}
	result := make([]domain.KubePersistentVolumeClaim, 0, len(list.Items))
	for _, pvc := range list.Items {
		capacity := ""
		if pvc.Status.Capacity != nil {
			if q, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
				capacity = q.String()
			}
		}
		sc := ""
		if pvc.Spec.StorageClassName != nil {
			sc = *pvc.Spec.StorageClassName
		}
		result = append(result, domain.KubePersistentVolumeClaim{
			Name:             pvc.Name,
			Namespace:        pvc.Namespace,
			Status:           string(pvc.Status.Phase),
			VolumeName:       pvc.Spec.VolumeName,
			Capacity:         capacity,
			AccessModes:      accessModesToString(pvc.Spec.AccessModes),
			StorageClassName: sc,
		})
	}
	return result, nil
}

// Compile-time assertion.
var _ domain.KubeClient = (*kubeClientImpl)(nil)

// ── helpers ──────────────────────────────────────────────────────────────────

func toKubeService(svc corev1.Service) domain.KubeService {
	ports := make([]domain.KubeServicePort, 0, len(svc.Spec.Ports))
	for _, p := range svc.Spec.Ports {
		ports = append(ports, domain.KubeServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: p.TargetPort.String(),
			NodePort:   p.NodePort,
			Protocol:   string(p.Protocol),
		})
	}
	return domain.KubeService{
		Name:      svc.Name,
		Namespace: svc.Namespace,
		Type:      string(svc.Spec.Type),
		ClusterIP: svc.Spec.ClusterIP,
		Ports:     ports,
	}
}

// parseTargetPort converts a string like "80" to intstr.FromInt32(80),
// or a named port like "http" to intstr.FromString("http").
func parseTargetPort(s string) intstr.IntOrString {
	if n, err := strconv.ParseInt(s, 10, 32); err == nil {
		return intstr.FromInt32(int32(n))
	}
	return intstr.FromString(s)
}

func accessModesToString(modes []corev1.PersistentVolumeAccessMode) string {
	if len(modes) == 0 {
		return ""
	}
	out := string(modes[0])
	for _, m := range modes[1:] {
		out += "," + string(m)
	}
	return out
}

