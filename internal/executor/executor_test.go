package executor_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brazier/brazier/internal/executor"
	pb "github.com/brazier/brazier/proto/gen"
)

func dispatch(commands ...string) *pb.JobDispatch {
	return &pb.JobDispatch{
		JobId: "test-job",
		RunId: "test-run",
		Spec:  &pb.JobSpec{Commands: commands},
	}
}

func TestRunSuccess(t *testing.T) {
	exec := executor.New(10 * time.Second)
	var lines []string
	sink := func(c *pb.LogChunk) { lines = append(lines, c.Line) }

	ok, code := exec.Run(context.Background(), dispatch("echo hello"), sink)
	if !ok || code != 0 {
		t.Errorf("run: ok=%v code=%d", ok, code)
	}
	if len(lines) == 0 || !strings.Contains(lines[0], "hello") {
		t.Errorf("lines = %v", lines)
	}
}

func TestRunFailure(t *testing.T) {
	exec := executor.New(10 * time.Second)
	ok, code := exec.Run(context.Background(), dispatch("exit 42"), func(*pb.LogChunk) {})
	if ok || code != 42 {
		t.Errorf("expected failure exit 42, got ok=%v code=%d", ok, code)
	}
}

func TestRunTimeout(t *testing.T) {
	exec := executor.New(100 * time.Millisecond)
	ok, _ := exec.Run(context.Background(), dispatch("sleep 10"), func(*pb.LogChunk) {})
	if ok {
		t.Error("expected timeout failure, got success")
	}
}

func TestRunMultipleCommands(t *testing.T) {
	exec := executor.New(10 * time.Second)
	var lines []string
	sink := func(c *pb.LogChunk) { lines = append(lines, c.Line) }

	ok, _ := exec.Run(context.Background(), dispatch("echo one", "echo two", "echo three"), sink)
	if !ok {
		t.Fatal("expected success")
	}
	combined := strings.Join(lines, " ")
	for _, want := range []string{"one", "two", "three"} {
		if !strings.Contains(combined, want) {
			t.Errorf("missing %q in output: %v", want, lines)
		}
	}
}

func TestRunStopsOnFirstFailure(t *testing.T) {
	exec := executor.New(10 * time.Second)
	var lines []string
	sink := func(c *pb.LogChunk) { lines = append(lines, c.Line) }

	ok, _ := exec.Run(context.Background(), dispatch("echo before", "exit 1", "echo after"), sink)
	if ok {
		t.Fatal("expected failure")
	}
	combined := strings.Join(lines, " ")
	if strings.Contains(combined, "after") {
		t.Error("should not have executed command after failure")
	}
}

func TestEnvVars(t *testing.T) {
	exec := executor.New(10 * time.Second)
	var lines []string
	sink := func(c *pb.LogChunk) { lines = append(lines, c.Line) }

	d := &pb.JobDispatch{
		JobId: "j",
		RunId: "r",
		Spec:  &pb.JobSpec{Commands: []string{"echo $BRAZIER_TEST_VAR"}},
		Env:   []*pb.EnvVar{{Key: "BRAZIER_TEST_VAR", Value: "hello-from-env"}},
	}
	ok, _ := exec.Run(context.Background(), d, sink)
	if !ok {
		t.Fatal("expected success")
	}
	if len(lines) == 0 || !strings.Contains(lines[0], "hello-from-env") {
		t.Errorf("env var not injected: %v", lines)
	}
}
