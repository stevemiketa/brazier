// Package pipeline manages pipeline run state and orchestrates the scheduler.
package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/brazier/brazier/internal/bus"
	"github.com/brazier/brazier/internal/db"
	pb "github.com/brazier/brazier/proto/gen"
	"github.com/google/uuid"
)

// RunContext carries metadata about the triggering event, used for condition evaluation.
type RunContext struct {
	Branch string
	Tag    string
	Event  string // e.g. "push", "pull_request"
}

// Manager orchestrates pipeline run lifecycle.
type Manager struct {
	db        db.DB
	bus       *bus.Bus
	scheduler *Scheduler
}

// NewManager returns a Manager wired to the given DB and event bus.
func NewManager(store db.DB, b *bus.Bus, scheduler *Scheduler) *Manager {
	return &Manager{db: store, bus: b, scheduler: scheduler}
}

// Start creates a new pipeline run from spec, seeds node records, and begins scheduling.
func (m *Manager) Start(ctx context.Context, spec *pb.PipelineSpec, project string, rc RunContext) (string, error) {
	runID := uuid.New().String()
	now := time.Now()

	run := db.PipelineRun{
		ID: runID, Project: project,
		State: db.RunStatePending, CreatedAt: now, UpdatedAt: now,
	}
	if err := m.db.CreateRun(ctx, run); err != nil {
		return "", fmt.Errorf("create run: %w", err)
	}

	// Seed all top-level nodes as pending.
	for _, node := range spec.Nodes {
		if err := m.db.UpsertNode(ctx, db.NodeRecord{
			RunID: runID, NodeID: node.Id, State: db.NodeStatePending,
		}); err != nil {
			return "", fmt.Errorf("seed node %s: %w", node.Id, err)
		}
		// For stages, seed inner jobs too.
		if s := node.GetStage(); s != nil {
			for _, j := range s.Jobs {
				if err := m.db.UpsertNode(ctx, db.NodeRecord{
					RunID: runID, NodeID: node.Id + "/" + j.Id, State: db.NodeStatePending,
				}); err != nil {
					return "", err
				}
			}
		}
	}

	if err := m.db.UpdateRunState(ctx, runID, db.RunStateRunning); err != nil {
		return "", err
	}

	m.bus.Publish(bus.Event{Type: bus.EventTriggerReceived, Payload: runID})

	slog.Info("run started", "run_id", runID, "project", project)
	go m.scheduler.Advance(context.Background(), runID, spec, rc)

	return runID, nil
}

// NodeCompleted is called by the agent registry when a job finishes.
func (m *Manager) NodeCompleted(ctx context.Context, runID, nodeID string, success bool) error {
	state := db.NodeStateSuccess
	evType := bus.EventJobCompleted
	if !success {
		state = db.NodeStateFailed
		evType = bus.EventJobFailed
	}

	if err := m.db.UpsertNode(ctx, db.NodeRecord{
		RunID: runID, NodeID: nodeID, State: state,
	}); err != nil {
		return err
	}

	m.bus.Publish(bus.Event{Type: evType, Payload: nodeID})

	// Let the scheduler decide what to dispatch next.
	nodes, err := m.db.ListNodes(ctx, runID)
	if err != nil {
		return err
	}
	if allTerminal(nodes) {
		finalState := db.RunStateSuccess
		for _, n := range nodes {
			if n.State == db.NodeStateFailed {
				finalState = db.RunStateFailed
				break
			}
		}
		if err := m.db.UpdateRunState(ctx, runID, finalState); err != nil {
			return err
		}
		m.bus.Publish(bus.Event{Type: bus.EventRunCompleted, Payload: runID})
		slog.Info("run completed", "run_id", runID, "state", finalState)
	}
	return nil
}

func allTerminal(nodes []db.NodeRecord) bool {
	for _, n := range nodes {
		if n.State == db.NodeStatePending || n.State == db.NodeStateRunning {
			return false
		}
	}
	return true
}
