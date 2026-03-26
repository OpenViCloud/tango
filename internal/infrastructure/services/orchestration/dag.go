package orchestration

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

type NodeMode string

const (
	ModeSequential NodeMode = "sequential"
	ModeParallel   NodeMode = "parallel"
)

type NodeConfig struct {
	ID        string
	AgentID   string
	DependsOn []string
	Mode      NodeMode
}

type WorkflowConfig struct {
	Nodes []NodeConfig
}

type dagNode struct {
	config   NodeConfig
	children []string
}

type DAG struct {
	nodes    map[string]*dagNode
	entries  []string
	topoSort []string
}

type NodeResult struct {
	NodeID  string
	AgentID string
	Input   string
	Output  string
}

type DAGExecutor struct {
	RunAgent func(ctx context.Context, agentID, input string) (string, error)
}

func BuildDAG(cfg WorkflowConfig) (*DAG, error) {
	if len(cfg.Nodes) == 0 {
		return nil, errors.New("dag: workflow must contain at least one node")
	}

	dag := &DAG{
		nodes: make(map[string]*dagNode, len(cfg.Nodes)),
	}

	for _, nc := range cfg.Nodes {
		if strings.TrimSpace(nc.ID) == "" {
			return nil, errors.New("dag: node id is required")
		}
		if strings.TrimSpace(nc.AgentID) == "" {
			return nil, fmt.Errorf("dag: node %q requires agent_id", nc.ID)
		}

		normalized := normalizeNodeConfig(nc)
		if _, exists := dag.nodes[normalized.ID]; exists {
			return nil, fmt.Errorf("dag: duplicate node id %q", normalized.ID)
		}
		dag.nodes[normalized.ID] = &dagNode{config: normalized}
	}

	for _, node := range dag.nodes {
		for _, dep := range node.config.DependsOn {
			dependencyNode, exists := dag.nodes[dep]
			if !exists {
				return nil, fmt.Errorf("dag: node %q depends on missing node %q", node.config.ID, dep)
			}
			dependencyNode.children = append(dependencyNode.children, node.config.ID)
		}
	}

	for id, node := range dag.nodes {
		if len(node.config.DependsOn) == 0 {
			dag.entries = append(dag.entries, id)
		}
	}
	slices.Sort(dag.entries)

	topoSort, err := kahnTopoSort(dag)
	if err != nil {
		return nil, err
	}
	dag.topoSort = topoSort

	if err := validateSiblingModes(dag); err != nil {
		return nil, err
	}

	return dag, nil
}

func (e *DAGExecutor) Execute(ctx context.Context, dag *DAG, initialInput string) ([]NodeResult, error) {
	if e == nil || e.RunAgent == nil {
		return nil, errors.New("dag: executor requires RunAgent")
	}
	if dag == nil {
		return nil, errors.New("dag: dag is required")
	}

	outputs := make(map[string]string, len(dag.nodes))
	processed := make(map[string]bool, len(dag.nodes))
	results := make([]NodeResult, 0, len(dag.nodes))
	var outputsMu sync.Mutex

	resolveInput := func(nodeID string) string {
		deps := dag.nodes[nodeID].config.DependsOn
		if len(deps) == 0 {
			return initialInput
		}
		if len(deps) == 1 {
			outputsMu.Lock()
			defer outputsMu.Unlock()
			return outputs[deps[0]]
		}

		outputsMu.Lock()
		defer outputsMu.Unlock()
		parts := make([]string, 0, len(deps))
		for _, dep := range deps {
			parts = append(parts, fmt.Sprintf("[%s]: %s", dep, outputs[dep]))
		}
		return strings.Join(parts, "\n\n")
	}

	for len(processed) < len(dag.nodes) {
		readyGroups := make(map[string][]string)
		for _, nodeID := range dag.topoSort {
			if processed[nodeID] {
				continue
			}

			node := dag.nodes[nodeID]
			allDepsReady := true
			for _, dep := range node.config.DependsOn {
				if !processed[dep] {
					allDepsReady = false
					break
				}
			}
			if !allDepsReady {
				continue
			}

			groupKey := dependencyGroupKey(node.config.DependsOn)
			readyGroups[groupKey] = append(readyGroups[groupKey], nodeID)
		}

		if len(readyGroups) == 0 {
			return nil, errors.New("dag: executor stuck with no ready nodes")
		}

		readyGroupKeys := make([]string, 0, len(readyGroups))
		for key := range readyGroups {
			readyGroupKeys = append(readyGroupKeys, key)
		}
		slices.Sort(readyGroupKeys)

		for _, groupKey := range readyGroupKeys {
			group := readyGroups[groupKey]
			if len(group) == 0 {
				continue
			}

			mode := dag.nodes[group[0]].config.Mode
			if mode == ModeParallel {
				groupResults, err := e.executeParallelGroup(ctx, dag, group, resolveInput)
				if err != nil {
					return nil, err
				}
				for _, result := range groupResults {
					outputs[result.NodeID] = result.Output
					processed[result.NodeID] = true
					results = append(results, result)
				}
				continue
			}

			for _, nodeID := range group {
				input := resolveInput(nodeID)
				output, err := e.RunAgent(ctx, dag.nodes[nodeID].config.AgentID, input)
				if err != nil {
					return nil, fmt.Errorf("dag: node %q failed: %w", nodeID, err)
				}

				result := NodeResult{
					NodeID:  nodeID,
					AgentID: dag.nodes[nodeID].config.AgentID,
					Input:   input,
					Output:  output,
				}
				outputsMu.Lock()
				outputs[nodeID] = output
				outputsMu.Unlock()
				processed[nodeID] = true
				results = append(results, result)
			}
		}
	}

	resultOrder := make(map[string]int, len(results))
	for idx, result := range results {
		resultOrder[result.NodeID] = idx
	}
	slices.SortStableFunc(results, func(a, b NodeResult) int {
		aTopo := slices.Index(dag.topoSort, a.NodeID)
		bTopo := slices.Index(dag.topoSort, b.NodeID)
		if aTopo != bTopo {
			return aTopo - bTopo
		}
		return resultOrder[a.NodeID] - resultOrder[b.NodeID]
	})

	return results, nil
}

func (e *DAGExecutor) executeParallelGroup(
	ctx context.Context,
	dag *DAG,
	group []string,
	resolveInput func(nodeID string) string,
) ([]NodeResult, error) {
	results := make([]NodeResult, len(group))
	g, gctx := errgroup.WithContext(ctx)

	for idx, nodeID := range group {
		idx := idx
		nodeID := nodeID
		g.Go(func() error {
			input := resolveInput(nodeID)
			output, err := e.RunAgent(gctx, dag.nodes[nodeID].config.AgentID, input)
			if err != nil {
				return fmt.Errorf("dag: node %q failed: %w", nodeID, err)
			}
			results[idx] = NodeResult{
				NodeID:  nodeID,
				AgentID: dag.nodes[nodeID].config.AgentID,
				Input:   input,
				Output:  output,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

func kahnTopoSort(dag *DAG) ([]string, error) {
	inDegree := make(map[string]int, len(dag.nodes))
	for id := range dag.nodes {
		inDegree[id] = 0
	}
	for _, node := range dag.nodes {
		for _, child := range node.children {
			inDegree[child]++
		}
	}

	queue := make([]string, 0, len(dag.entries))
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	slices.Sort(queue)

	result := make([]string, 0, len(dag.nodes))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		children := append([]string(nil), dag.nodes[current].children...)
		slices.Sort(children)
		for _, child := range children {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
				slices.Sort(queue)
			}
		}
	}

	if len(result) != len(dag.nodes) {
		cycleNodes := make([]string, 0)
		visited := make(map[string]struct{}, len(result))
		for _, id := range result {
			visited[id] = struct{}{}
		}
		for id := range dag.nodes {
			if _, ok := visited[id]; !ok {
				cycleNodes = append(cycleNodes, id)
			}
		}
		slices.Sort(cycleNodes)
		return nil, fmt.Errorf("dag: cycle detected across nodes [%s]", strings.Join(cycleNodes, ", "))
	}

	return result, nil
}

func validateSiblingModes(dag *DAG) error {
	groups := make(map[string][]NodeConfig)
	for _, node := range dag.nodes {
		key := dependencyGroupKey(node.config.DependsOn)
		groups[key] = append(groups[key], node.config)
	}

	for _, group := range groups {
		if len(group) <= 1 {
			continue
		}

		parallelCount := 0
		for _, node := range group {
			if node.Mode == ModeParallel {
				parallelCount++
			}
		}

		if parallelCount == 0 || parallelCount == len(group) {
			continue
		}

		nodeIDs := make([]string, 0, len(group))
		for _, node := range group {
			nodeIDs = append(nodeIDs, node.ID)
		}
		slices.Sort(nodeIDs)
		return fmt.Errorf(
			"dag: sibling group [%s] must be all sequential or all parallel",
			strings.Join(nodeIDs, ", "),
		)
	}

	return nil
}

func normalizeNodeConfig(node NodeConfig) NodeConfig {
	node.ID = strings.TrimSpace(node.ID)
	node.AgentID = strings.TrimSpace(node.AgentID)
	node.DependsOn = normalizeDependsOn(node.DependsOn)
	switch node.Mode {
	case ModeParallel:
		return node
	default:
		node.Mode = ModeSequential
		return node
	}
}

func normalizeDependsOn(dependsOn []string) []string {
	if len(dependsOn) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(dependsOn))
	normalized := make([]string, 0, len(dependsOn))
	for _, dep := range dependsOn {
		value := strings.TrimSpace(dep)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	slices.Sort(normalized)
	return normalized
}

func dependencyGroupKey(dependsOn []string) string {
	if len(dependsOn) == 0 {
		return "__entry__"
	}
	return strings.Join(dependsOn, "|")
}
