// Package db provides the database interface and implementations for the
// master service.
package db

import (
	"context"
	"time"
)

// RunState represents the lifecycle state of a pipeline run.
type RunState string

const (
	RunStatePending   RunState = "pending"
	RunStateRunning   RunState = "running"
	RunStateSuccess   RunState = "success"
	RunStateFailed    RunState = "failed"
	RunStateCancelled RunState = "cancelled"
)

// NodeState represents the lifecycle state of a single DAG node within a run.
type NodeState string

const (
	NodeStatePending   NodeState = "pending"
	NodeStateRunning   NodeState = "running"
	NodeStateSuccess   NodeState = "success"
	NodeStateFailed    NodeState = "failed"
	NodeStateSkipped   NodeState = "skipped"
)

// PipelineRun is the persisted record of a single run.
type PipelineRun struct {
	ID        string
	Project   string
	State     RunState
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NodeRecord is the persisted state of a DAG node within a run.
type NodeRecord struct {
	RunID     string
	NodeID    string
	JobID     string // assigned when dispatched
	State     NodeState
	UpdatedAt time.Time
}

// LogChunk is a single line of log output from a job.
type LogChunk struct {
	JobID     string
	RunID     string
	Timestamp int64
	Line      string
	Stderr    bool
}

// DB is the storage interface used by the master service.
type DB interface {
	// Runs
	CreateRun(ctx context.Context, run PipelineRun) error
	GetRun(ctx context.Context, id string) (PipelineRun, error)
	UpdateRunState(ctx context.Context, id string, state RunState) error
	ListRuns(ctx context.Context, project string, limit int) ([]PipelineRun, error)

	// Nodes
	UpsertNode(ctx context.Context, node NodeRecord) error
	GetNode(ctx context.Context, runID, nodeID string) (NodeRecord, error)
	ListNodes(ctx context.Context, runID string) ([]NodeRecord, error)

	// Logs
	AppendLog(ctx context.Context, chunk LogChunk) error
	GetLogs(ctx context.Context, runID, jobID string) ([]LogChunk, error)

	// Lifecycle
	Close() error
}
