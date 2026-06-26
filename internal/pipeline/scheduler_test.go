package pipeline_test

import (
	"context"
	"testing"

	"github.com/brazier/brazier/internal/pipeline"
	pb "github.com/brazier/brazier/proto/gen"
)

func spec(nodes ...*pb.Node) *pb.PipelineSpec {
	return &pb.PipelineSpec{Nodes: nodes}
}

func jobNode(id string, deps ...string) *pb.Node {
	return &pb.Node{
		Id:        id,
		DependsOn: deps,
		Kind:      &pb.Node_Job{Job: &pb.JobSpec{Commands: []string{"echo " + id}}},
	}
}

func TestTopoSortLinear(t *testing.T) {
	s := spec(jobNode("build", "test"), jobNode("test", "lint"), jobNode("lint"))
	sorted, err := pipeline.TopoSort(s)
	if err != nil {
		t.Fatalf("topo sort: %v", err)
	}
	order := make([]string, len(sorted))
	for i, n := range sorted {
		order[i] = n.Id
	}
	// lint must come before test, test before build
	pos := func(id string) int {
		for i, n := range sorted {
			if n.Id == id {
				return i
			}
		}
		return -1
	}
	if pos("lint") >= pos("test") || pos("test") >= pos("build") {
		t.Errorf("wrong order: %v", order)
	}
}

func TestTopoSortCycle(t *testing.T) {
	s := spec(jobNode("a", "b"), jobNode("b", "a"))
	if _, err := pipeline.TopoSort(s); err == nil {
		t.Error("expected cycle error, got nil")
	}
}

func TestConditionEvalBranch(t *testing.T) {
	rc := pipeline.RunContext{Branch: "main"}
	cases := []struct {
		conds []string
		want  bool
	}{
		{[]string{"branch == main"}, true},
		{[]string{"branch == dev"}, false},
		{[]string{}, true},
		{[]string{"event == push", "branch == main"}, false}, // event unset
	}
	for _, c := range cases {
		node := &pb.Node{
			Id:         "n",
			Conditions: c.conds,
			Kind:       &pb.Node_Job{Job: &pb.JobSpec{}},
		}
		got := pipeline.NodePassesConditions(node, rc)
		if got != c.want {
			t.Errorf("conditions %v: got %v, want %v", c.conds, got, c.want)
		}
	}
}

type fakeDispatcher struct {
	dispatched []string
}

func (f *fakeDispatcher) Dispatch(_ context.Context, _, nodeID string, _ *pb.JobSpec) error {
	f.dispatched = append(f.dispatched, nodeID)
	return nil
}

func TestSchedulerAdvanceDispatches(t *testing.T) {
	store := openDB(t)
	ctx := context.Background()
	fd := &fakeDispatcher{}
	b := newBus()
	sched := pipeline.NewScheduler(store, fd, b)

	runID := seedRun(t, store, ctx, spec(
		jobNode("lint"),
		jobNode("build", "lint"),
	))

	pspec := spec(jobNode("lint"), jobNode("build", "lint"))
	sched.Advance(ctx, runID, pspec, pipeline.RunContext{})

	if len(fd.dispatched) != 1 || fd.dispatched[0] != "lint" {
		t.Errorf("dispatched = %v, want [lint]", fd.dispatched)
	}
}
