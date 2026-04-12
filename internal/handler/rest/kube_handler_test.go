package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tango/internal/domain"
	"tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeKubeClient struct {
	pods       []domain.KubePod
	services   []domain.KubeService
	createdPod *domain.KubePod
	createdSvc *domain.KubeService
	deletedPod string
	deletedSvc string
	err        error
}

func (f *fakeKubeClient) ListNamespaces(_ context.Context) ([]domain.KubeNamespace, error) {
	return nil, f.err
}
func (f *fakeKubeClient) ListPods(_ context.Context, _ string) ([]domain.KubePod, error) {
	return f.pods, f.err
}
func (f *fakeKubeClient) CreatePod(_ context.Context, namespace string, spec domain.KubePodSpec) (*domain.KubePod, error) {
	if f.err != nil {
		return nil, f.err
	}
	pod := &domain.KubePod{Name: spec.Name, Namespace: namespace, Status: "Pending"}
	f.createdPod = pod
	return pod, nil
}
func (f *fakeKubeClient) DeletePod(_ context.Context, _ string, name string) error {
	f.deletedPod = name
	return f.err
}
func (f *fakeKubeClient) ListServices(_ context.Context, _ string) ([]domain.KubeService, error) {
	return f.services, f.err
}
func (f *fakeKubeClient) CreateService(_ context.Context, namespace string, spec domain.KubeServiceSpec) (*domain.KubeService, error) {
	if f.err != nil {
		return nil, f.err
	}
	svc := &domain.KubeService{Name: spec.Name, Namespace: namespace, Type: spec.Type}
	f.createdSvc = svc
	return svc, nil
}
func (f *fakeKubeClient) DeleteService(_ context.Context, _ string, name string) error {
	f.deletedSvc = name
	return f.err
}
func (f *fakeKubeClient) ListPersistentVolumes(_ context.Context) ([]domain.KubePersistentVolume, error) {
	return nil, f.err
}
func (f *fakeKubeClient) ListPersistentVolumeClaims(_ context.Context, _ string) ([]domain.KubePersistentVolumeClaim, error) {
	return nil, f.err
}
func (f *fakeKubeClient) CreateOrUpdateSecret(_ context.Context, _, _ string, _ map[string]string) error {
	return f.err
}
func (f *fakeKubeClient) CreateOrUpdateConfigMap(_ context.Context, _, _ string, _ map[string]string) error {
	return f.err
}
func (f *fakeKubeClient) CreateOrUpdateDeployment(_ context.Context, _ string, _ domain.KubeDeploymentSpec) error {
	return f.err
}
func (f *fakeKubeClient) DeleteDeployment(_ context.Context, _, _ string) error {
	return f.err
}
func (f *fakeKubeClient) RolloutRestartDeployment(_ context.Context, _, _ string) error {
	return f.err
}

var _ domain.KubeClient = (*fakeKubeClient)(nil)

// fakeKubeFactory always returns the same fakeKubeClient.
type fakeKubeFactory struct {
	client *fakeKubeClient
	err    error
}

func (f *fakeKubeFactory) GetClient(_ context.Context, _ string) (domain.KubeClient, error) {
	return f.client, f.err
}
func (f *fakeKubeFactory) InvalidateClient(_ string) {}

var _ domain.KubeClientFactory = (*fakeKubeFactory)(nil)

// fakeClusterRepoForKube returns a ready cluster by default.
type fakeClusterRepoForKube struct {
	cluster *domain.Cluster
	err     error
}

func (f *fakeClusterRepoForKube) GetByID(_ context.Context, _ string) (*domain.Cluster, error) {
	return f.cluster, f.err
}
func (f *fakeClusterRepoForKube) Save(_ context.Context, c *domain.Cluster) (*domain.Cluster, error) {
	return c, nil
}
func (f *fakeClusterRepoForKube) Update(_ context.Context, c *domain.Cluster) (*domain.Cluster, error) {
	return c, nil
}
func (f *fakeClusterRepoForKube) UpdateStatus(_ context.Context, _ string, _ domain.ClusterStatus, _ string) error {
	return nil
}
func (f *fakeClusterRepoForKube) UpdateKubeconfig(_ context.Context, _ string, _ string) error {
	return nil
}
func (f *fakeClusterRepoForKube) ListAll(_ context.Context) ([]*domain.Cluster, error) {
	return nil, nil
}
func (f *fakeClusterRepoForKube) Delete(_ context.Context, _ string) error { return nil }

var _ domain.ClusterRepository = (*fakeClusterRepoForKube)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func readyCluster() *domain.Cluster {
	return &domain.Cluster{
		ID:        "cluster-1",
		Name:      "test",
		Status:    domain.ClusterStatusReady,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newKubeTestRouter(client *fakeKubeClient, clusterErr error) *gin.Engine {
	gin.SetMode(gin.TestMode)

	cluster := readyCluster()
	if clusterErr != nil {
		cluster = nil
	}

	factory := &fakeKubeFactory{client: client}
	repo := &fakeClusterRepoForKube{cluster: cluster, err: clusterErr}

	r := gin.New()
	r.Use(response.Middleware(nil))
	NewKubeHandler(factory, repo).Register(r.Group("/"))
	return r
}

func postJSON(t *testing.T, r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// ── Create Pod ────────────────────────────────────────────────────────────────

func TestKubeHandler_CreatePod_OK(t *testing.T) {
	client := &fakeKubeClient{}
	r := newKubeTestRouter(client, nil)

	w := postJSON(t, r, "/clusters/cluster-1/pods?namespace=default", map[string]any{
		"name":  "nginx",
		"image": "nginx:latest",
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body)
	}
	if client.createdPod == nil {
		t.Fatal("expected pod to be created")
	}
	if client.createdPod.Name != "nginx" {
		t.Fatalf("pod name = %q, want nginx", client.createdPod.Name)
	}
}

func TestKubeHandler_CreatePod_MissingFields(t *testing.T) {
	r := newKubeTestRouter(&fakeKubeClient{}, nil)

	w := postJSON(t, r, "/clusters/cluster-1/pods", map[string]any{
		"name": "nginx",
		// image missing
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestKubeHandler_CreatePod_ClusterNotReady(t *testing.T) {
	r := newKubeTestRouter(&fakeKubeClient{}, domain.ErrClusterNotFound)

	w := postJSON(t, r, "/clusters/cluster-1/pods", map[string]any{
		"name":  "nginx",
		"image": "nginx:latest",
	})

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

// ── Delete Pod ────────────────────────────────────────────────────────────────

func TestKubeHandler_DeletePod_OK(t *testing.T) {
	client := &fakeKubeClient{}
	r := newKubeTestRouter(client, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/clusters/cluster-1/pods/nginx?namespace=default", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body: %s", w.Code, w.Body)
	}
	if client.deletedPod != "nginx" {
		t.Fatalf("deletedPod = %q, want nginx", client.deletedPod)
	}
}

func TestKubeHandler_DeletePod_ClusterNotFound(t *testing.T) {
	r := newKubeTestRouter(&fakeKubeClient{}, domain.ErrClusterNotFound)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/clusters/cluster-1/pods/nginx?namespace=default", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

// ── Create Service ────────────────────────────────────────────────────────────

func TestKubeHandler_CreateService_OK(t *testing.T) {
	client := &fakeKubeClient{}
	r := newKubeTestRouter(client, nil)

	w := postJSON(t, r, "/clusters/cluster-1/services?namespace=default", map[string]any{
		"name":     "nginx-svc",
		"type":     "ClusterIP",
		"selector": map[string]string{"app": "nginx"},
		"ports": []map[string]any{
			{"name": "http", "port": 80, "target_port": "80", "protocol": "TCP"},
		},
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body)
	}
	if client.createdSvc == nil {
		t.Fatal("expected service to be created")
	}
	if client.createdSvc.Name != "nginx-svc" {
		t.Fatalf("svc name = %q, want nginx-svc", client.createdSvc.Name)
	}
}

func TestKubeHandler_CreateService_MissingFields(t *testing.T) {
	r := newKubeTestRouter(&fakeKubeClient{}, nil)

	w := postJSON(t, r, "/clusters/cluster-1/services", map[string]any{
		"name": "nginx-svc",
		// type, selector, ports missing
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// ── Delete Service ────────────────────────────────────────────────────────────

func TestKubeHandler_DeleteService_OK(t *testing.T) {
	client := &fakeKubeClient{}
	r := newKubeTestRouter(client, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/clusters/cluster-1/services/nginx-svc?namespace=default", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body: %s", w.Code, w.Body)
	}
	if client.deletedSvc != "nginx-svc" {
		t.Fatalf("deletedSvc = %q, want nginx-svc", client.deletedSvc)
	}
}

func TestKubeHandler_DeleteService_ClusterNotFound(t *testing.T) {
	r := newKubeTestRouter(&fakeKubeClient{}, domain.ErrClusterNotFound)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/clusters/cluster-1/services/nginx-svc?namespace=default", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
