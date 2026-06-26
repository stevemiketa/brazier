package pipeline_test

import (
	"context"
	"testing"
	"time"

	"github.com/brazier/brazier/internal/bus"
	"github.com/brazier/brazier/internal/db"
	pb "github.com/brazier/brazier/proto/gen"
)

func openDB(t *testing.T) db.DB {
	t.Helper()
	s, err := db.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func newBus() *bus.Bus { return bus.New() }

func seedRun(t *testing.T, store db.DB, ctx context.Context, s *pb.PipelineSpec) string {
	t.Helper()
	runID := "test-run-1"
	now := time.Now()
	if err := store.CreateRun(ctx, db.PipelineRun{
		ID: runID, Project: "test", State: db.RunStatePending,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	for _, n := range s.Nodes {
		if err := store.UpsertNode(ctx, db.NodeRecord{
			RunID: runID, NodeID: n.Id, State: db.NodeStatePending,
		}); err != nil {
			t.Fatalf("seed node: %v", err)
		}
	}
	return runID
}
