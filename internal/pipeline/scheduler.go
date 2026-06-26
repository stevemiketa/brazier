package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/brazier/brazier/internal/bus"
	"github.com/brazier/brazier/internal/db"
	pb "github.com/brazier/brazier/proto/gen"
)

// Dispatcher sends a ready job to an agent. Implemented by the agent registry.
type Dispatcher interface {
	Dispatch(ctx context.Context, runID, nodeID string, spec *pb.JobSpec) error
}

// Scheduler performs topological ordering and dispatches ready nodes.
// On node failure, unrelated branches continue (continue-on-failure strategy).
type Scheduler struct {
	store      db.DB
	dispatcher Dispatcher
	bus        *bus.Bus
}

// NewScheduler returns a Scheduler wired to the given dependencies.
func NewScheduler(store db.DB, d Dispatcher, b *bus.Bus) *Scheduler {
	return &Scheduler{store: store, dispatcher: d, bus: b}
}

// Advance evaluates the current DAG state for runID and dispatches any newly
// unblocked nodes. It is safe to call concurrently; each call is idempotent.
func (s *Scheduler) Advance(ctx context.Context, runID string, spec *pb.PipelineSpec, rc RunContext) {
	nodes, err := s.store.ListNodes(ctx, runID)
	if err != nil {
		slog.Error("scheduler: list nodes", "err", err)
		return
	}

	stateByID := make(map[string]db.NodeState, len(nodes))
	for _, n := range nodes {
		stateByID[n.NodeID] = n.State
	}

	for _, node := range spec.Nodes {
		if stateByID[node.Id] != db.NodeStatePending {
			continue
		}
		if !depsTerminal(node.DependsOn, stateByID) {
			continue
		}
		if !evalConditions(node.Conditions, rc) {
			if err := s.store.UpsertNode(ctx, db.NodeRecord{
				RunID: runID, NodeID: node.Id, State: db.NodeStateSkipped,
			}); err != nil {
				slog.Error("scheduler: skip node", "node", node.Id, "err", err)
			}
			continue
		}

		if err := s.store.UpsertNode(ctx, db.NodeRecord{
			RunID: runID, NodeID: node.Id, State: db.NodeStateRunning,
		}); err != nil {
			slog.Error("scheduler: mark running", "node", node.Id, "err", err)
			continue
		}

		switch k := node.Kind.(type) {
		case *pb.Node_Job:
			s.dispatch(ctx, runID, node.Id, k.Job)
		case *pb.Node_Stage:
			for _, j := range k.Stage.Jobs {
				s.dispatch(ctx, runID, node.Id+"/"+j.Id, j.GetJob())
			}
		}
	}
}

func (s *Scheduler) dispatch(ctx context.Context, runID, nodeID string, spec *pb.JobSpec) {
	if err := s.dispatcher.Dispatch(ctx, runID, nodeID, spec); err != nil {
		slog.Error("scheduler: dispatch", "node", nodeID, "err", err)
		return
	}
	s.bus.Publish(bus.Event{Type: bus.EventJobDispatched, Payload: nodeID})
}

// depsTerminal returns true when all dependency nodes have reached a terminal state
// (success or skipped). A failed dependency blocks the node — the DAG continues
// on other branches but won't execute nodes downstream of a failure.
func depsTerminal(deps []string, state map[string]db.NodeState) bool {
	for _, dep := range deps {
		switch state[dep] {
		case db.NodeStateSuccess, db.NodeStateSkipped:
			// ok
		default:
			return false
		}
	}
	return true
}

// NodePassesConditions is the exported form of evalConditions for testing.
func NodePassesConditions(node *pb.Node, rc RunContext) bool {
	return evalConditions(node.Conditions, rc)
}

// evalConditions returns true if all conditions pass for the given run context.
// Conditions use a simple "key == value" format serialized by the SDK.
func evalConditions(conditions []string, rc RunContext) bool {
	for _, cond := range conditions {
		if !evalCondition(cond, rc) {
			return false
		}
	}
	return true
}

func evalCondition(cond string, rc RunContext) bool {
	parts := strings.SplitN(cond, " == ", 2)
	if len(parts) != 2 {
		return false
	}
	key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	switch key {
	case "branch":
		return rc.Branch == val
	case "tag":
		return matchGlob(val, rc.Tag)
	case "event":
		return rc.Event == val
	default:
		slog.Warn("unknown condition key", "key", key)
		return false
	}
}

// matchGlob does simple prefix/suffix wildcard matching (e.g. "v*").
func matchGlob(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, strings.TrimSuffix(pattern, "*"))
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, strings.TrimPrefix(pattern, "*"))
	}
	return pattern == s
}

// TopoSort returns the nodes of spec in a valid topological order, or an error
// if the graph has cycles.
func TopoSort(spec *pb.PipelineSpec) ([]*pb.Node, error) {
	byID := make(map[string]*pb.Node, len(spec.Nodes))
	for _, n := range spec.Nodes {
		byID[n.Id] = n
	}

	var sorted []*pb.Node
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var visit func(id string) error
	visit = func(id string) error {
		if visited[id] {
			return nil
		}
		if inStack[id] {
			return fmt.Errorf("cycle detected at node %q", id)
		}
		inStack[id] = true
		node, ok := byID[id]
		if !ok {
			return fmt.Errorf("unknown node %q", id)
		}
		for _, dep := range node.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		delete(inStack, id)
		visited[id] = true
		sorted = append(sorted, node)
		return nil
	}

	for _, n := range spec.Nodes {
		if err := visit(n.Id); err != nil {
			return nil, err
		}
	}
	return sorted, nil
}
