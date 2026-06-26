package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/brazier/brazier/internal/db"
)

func openTestDB(t *testing.T) db.DB {
	t.Helper()
	s, err := db.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestRunCRUD(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	run := db.PipelineRun{
		ID: "run-1", Project: "myproject",
		State: db.RunStatePending, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := store.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != db.RunStatePending {
		t.Errorf("state = %q", got.State)
	}

	if err := store.UpdateRunState(ctx, "run-1", db.RunStateRunning); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = store.GetRun(ctx, "run-1")
	if got.State != db.RunStateRunning {
		t.Errorf("updated state = %q", got.State)
	}
}

func TestListRuns(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()
	now := time.Now()
	for _, id := range []string{"r1", "r2", "r3"} {
		_ = store.CreateRun(ctx, db.PipelineRun{
			ID: id, Project: "proj", State: db.RunStatePending,
			CreatedAt: now, UpdatedAt: now,
		})
	}
	runs, err := store.ListRuns(ctx, "proj", 2)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("got %d runs, want 2", len(runs))
	}
}

func TestNodeUpsert(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()
	now := time.Now()
	_ = store.CreateRun(ctx, db.PipelineRun{
		ID: "run-1", Project: "p", State: db.RunStatePending,
		CreatedAt: now, UpdatedAt: now,
	})

	n := db.NodeRecord{RunID: "run-1", NodeID: "lint", State: db.NodeStatePending}
	if err := store.UpsertNode(ctx, n); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	n.State = db.NodeStateRunning
	n.JobID = "job-abc"
	if err := store.UpsertNode(ctx, n); err != nil {
		t.Fatalf("upsert update: %v", err)
	}

	got, err := store.GetNode(ctx, "run-1", "lint")
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	if got.State != db.NodeStateRunning {
		t.Errorf("state = %q", got.State)
	}
	if got.JobID != "job-abc" {
		t.Errorf("job_id = %q", got.JobID)
	}
}

func TestLogChunks(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	chunks := []db.LogChunk{
		{JobID: "j1", RunID: "r1", Timestamp: 1, Line: "hello", Stderr: false},
		{JobID: "j1", RunID: "r1", Timestamp: 2, Line: "world", Stderr: true},
	}
	for _, c := range chunks {
		if err := store.AppendLog(ctx, c); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	got, err := store.GetLogs(ctx, "r1", "j1")
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d chunks, want 2", len(got))
	}
	if got[1].Stderr != true {
		t.Errorf("stderr flag wrong")
	}
}
