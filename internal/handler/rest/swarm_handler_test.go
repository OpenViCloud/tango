package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tango/internal/domain"

	"github.com/gin-gonic/gin"
)

// fakeSwarmRepository is a test double for domain.SwarmRepository.
type fakeSwarmRepository struct {
	isManager bool
	nodes     []domain.SwarmNode
	nodesErr  error
}

func (f *fakeSwarmRepository) IsManager(_ context.Context) bool { return f.isManager }

func (f *fakeSwarmRepository) CreateService(_ context.Context, _ domain.CreateServiceInput) (domain.SwarmService, error) {
	return domain.SwarmService{}, nil
}

func (f *fakeSwarmRepository) RemoveService(_ context.Context, _ string) error { return nil }

func (f *fakeSwarmRepository) EnsureOverlayNetwork(_ context.Context, _ string) error { return nil }

func (f *fakeSwarmRepository) ListNodes(_ context.Context) ([]domain.SwarmNode, error) {
	return f.nodes, f.nodesErr
}

var _ domain.SwarmRepository = (*fakeSwarmRepository)(nil)

func newSwarmTestRouter(repo domain.SwarmRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewSwarmHandler(repo).RegisterRoutes(r.Group("/"))
	return r
}

// ── GetStatus ────────────────────────────────────────────────────────────────

func TestSwarmHandler_GetStatus_NilRepo(t *testing.T) {
	r := newSwarmTestRouter(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swarm/status", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp swarmStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.IsManager {
		t.Fatal("IsManager = true, want false when repo is nil")
	}
}

func TestSwarmHandler_GetStatus_NotManager(t *testing.T) {
	r := newSwarmTestRouter(&fakeSwarmRepository{isManager: false})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swarm/status", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp swarmStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.IsManager {
		t.Fatal("IsManager = true, want false")
	}
}

func TestSwarmHandler_GetStatus_IsManager(t *testing.T) {
	r := newSwarmTestRouter(&fakeSwarmRepository{isManager: true})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swarm/status", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp swarmStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.IsManager {
		t.Fatal("IsManager = false, want true")
	}
}

// ── ListNodes ────────────────────────────────────────────────────────────────

func TestSwarmHandler_ListNodes_NotManager(t *testing.T) {
	r := newSwarmTestRouter(&fakeSwarmRepository{isManager: false})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swarm/nodes", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestSwarmHandler_ListNodes_ReturnsNodes(t *testing.T) {
	nodes := []domain.SwarmNode{
		{ID: "abc123", Hostname: "worker-1", Role: "worker", State: "ready", Availability: "active"},
		{ID: "def456", Hostname: "manager-1", Role: "manager", State: "ready", Availability: "active", ManagerAddr: "1.2.3.4:2377"},
	}
	r := newSwarmTestRouter(&fakeSwarmRepository{isManager: true, nodes: nodes})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swarm/nodes", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp []swarmNodeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("len(nodes) = %d, want 2", len(resp))
	}
	if resp[0].ID != "abc123" || resp[0].Hostname != "worker-1" {
		t.Fatalf("unexpected first node: %+v", resp[0])
	}
	if resp[1].ManagerAddr != "1.2.3.4:2377" {
		t.Fatalf("manager_addr = %q, want 1.2.3.4:2377", resp[1].ManagerAddr)
	}
}

func TestSwarmHandler_ListNodes_EmptyCluster(t *testing.T) {
	r := newSwarmTestRouter(&fakeSwarmRepository{isManager: true, nodes: []domain.SwarmNode{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swarm/nodes", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp []swarmNodeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("len(nodes) = %d, want 0", len(resp))
	}
}
