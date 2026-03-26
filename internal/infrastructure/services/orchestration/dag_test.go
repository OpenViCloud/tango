package orchestration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestBuildDAG_ValidParallelBranch(t *testing.T) {
	cfg := WorkflowConfig{
		Nodes: []NodeConfig{
			{ID: "research", AgentID: "agent-research"},
			{ID: "image", AgentID: "agent-image", DependsOn: []string{"research"}, Mode: ModeParallel},
			{ID: "post", AgentID: "agent-post", DependsOn: []string{"research"}, Mode: ModeParallel},
			{ID: "publish", AgentID: "agent-publish", DependsOn: []string{"image", "post"}},
		},
	}

	dag, err := BuildDAG(cfg)
	if err != nil {
		t.Fatalf("expected valid dag, got error: %v", err)
	}

	if len(dag.topoSort) != 4 {
		t.Fatalf("expected 4 nodes in topo sort, got %d", len(dag.topoSort))
	}
	if dag.topoSort[0] != "research" {
		t.Fatalf("expected research to be first, got %q", dag.topoSort[0])
	}
	if dag.topoSort[len(dag.topoSort)-1] != "publish" {
		t.Fatalf("expected publish to be last, got %q", dag.topoSort[len(dag.topoSort)-1])
	}
}

func TestBuildDAG_RejectsMixedSiblingModes(t *testing.T) {
	cfg := WorkflowConfig{
		Nodes: []NodeConfig{
			{ID: "A", AgentID: "agent-a"},
			{ID: "B", AgentID: "agent-b", DependsOn: []string{"A"}, Mode: ModeSequential},
			{ID: "C", AgentID: "agent-c", DependsOn: []string{"A"}, Mode: ModeParallel},
		},
	}

	_, err := BuildDAG(cfg)
	if err == nil || !strings.Contains(err.Error(), "all sequential or all parallel") {
		t.Fatalf("expected sibling mode validation error, got: %v", err)
	}
}

func TestBuildDAG_RejectsCycle(t *testing.T) {
	cfg := WorkflowConfig{
		Nodes: []NodeConfig{
			{ID: "A", AgentID: "agent-a", DependsOn: []string{"C"}},
			{ID: "B", AgentID: "agent-b", DependsOn: []string{"A"}},
			{ID: "C", AgentID: "agent-c", DependsOn: []string{"B"}},
		},
	}

	_, err := BuildDAG(cfg)
	if err == nil || !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("expected cycle error, got: %v", err)
	}
}

func TestExecute_DefaultSequentialWithinSiblingGroup(t *testing.T) {
	cfg := WorkflowConfig{
		Nodes: []NodeConfig{
			{ID: "A", AgentID: "agent-a"},
			{ID: "B", AgentID: "agent-b", DependsOn: []string{"A"}},
			{ID: "C", AgentID: "agent-c", DependsOn: []string{"A"}},
		},
	}

	dag, err := BuildDAG(cfg)
	if err != nil {
		t.Fatalf("build dag failed: %v", err)
	}

	callOrder := make([]string, 0, 3)
	var callMu sync.Mutex
	executor := &DAGExecutor{
		RunAgent: func(ctx context.Context, agentID, input string) (string, error) {
			callMu.Lock()
			callOrder = append(callOrder, agentID)
			callMu.Unlock()
			return fmt.Sprintf("output-%s", agentID), nil
		},
	}

	results, err := executor.Execute(context.Background(), dag, "start")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	expectedOrder := []string{"agent-a", "agent-b", "agent-c"}
	for idx, item := range expectedOrder {
		if callOrder[idx] != item {
			t.Fatalf("expected call order %v, got %v", expectedOrder, callOrder)
		}
	}
}

func TestExecute_ParallelBranchJoinsDependencyOutputs(t *testing.T) {
	cfg := WorkflowConfig{
		Nodes: []NodeConfig{
			{ID: "research", AgentID: "agent-research"},
			{ID: "image", AgentID: "agent-image", DependsOn: []string{"research"}, Mode: ModeParallel},
			{ID: "post", AgentID: "agent-post", DependsOn: []string{"research"}, Mode: ModeParallel},
			{ID: "publish", AgentID: "agent-publish", DependsOn: []string{"image", "post"}},
		},
	}

	dag, err := BuildDAG(cfg)
	if err != nil {
		t.Fatalf("build dag failed: %v", err)
	}

	executor := &DAGExecutor{
		RunAgent: func(ctx context.Context, agentID, input string) (string, error) {
			return fmt.Sprintf("output-of-%s(input=%s)", agentID, input), nil
		},
	}

	results, err := executor.Execute(context.Background(), dag, "topic: AI")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	var publish *NodeResult
	for idx := range results {
		if results[idx].NodeID == "publish" {
			publish = &results[idx]
			break
		}
	}
	if publish == nil {
		t.Fatal("expected publish result")
	}
	if !strings.Contains(publish.Input, "[image]:") || !strings.Contains(publish.Input, "[post]:") {
		t.Fatalf("expected publish input to contain both dependency outputs, got: %s", publish.Input)
	}
}
