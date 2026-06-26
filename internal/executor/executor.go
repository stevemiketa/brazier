// Package executor runs jobs as raw OS processes and streams log output.
package executor

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"

	pb "github.com/brazier/brazier/proto/gen"
)

// LogSink receives log lines as they are produced.
type LogSink func(chunk *pb.LogChunk)

// Executor runs a single JobDispatch as a sequence of shell commands.
type Executor struct {
	timeout time.Duration
}

// New returns an Executor with the given per-job timeout.
func New(timeout time.Duration) *Executor {
	return &Executor{timeout: timeout}
}

// Run executes each command in job.Spec.Commands sequentially.
// It calls sink for every line of stdout/stderr and returns success/exit code.
// The job is cancelled if ctx is done or the timeout fires.
func (e *Executor) Run(ctx context.Context, job *pb.JobDispatch, sink LogSink) (success bool, exitCode int) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	env := buildEnv(job)

	for _, command := range job.Spec.Commands {
		ok, code := e.runCommand(ctx, job, command, env, sink)
		if !ok {
			return false, code
		}
	}
	return true, 0
}

func (e *Executor) runCommand(ctx context.Context, job *pb.JobDispatch, command string, env []string, sink LogSink) (bool, int) {
	slog.Info("exec", "job", job.JobId, "cmd", command)

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Env = env

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		sink(errChunk(job, fmt.Sprintf("stdout pipe: %v", err)))
		return false, 1
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		sink(errChunk(job, fmt.Sprintf("stderr pipe: %v", err)))
		return false, 1
	}

	if err := cmd.Start(); err != nil {
		sink(errChunk(job, fmt.Sprintf("start: %v", err)))
		return false, 1
	}

	var wg sync.WaitGroup
	stream := func(pipe interface{ Read([]byte) (int, error) }, stderr bool) {
		defer wg.Done()
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			sink(&pb.LogChunk{
				JobId:     job.JobId,
				RunId:     job.RunId,
				Timestamp: time.Now().UnixMilli(),
				Line:      scanner.Text(),
				Stderr:    stderr,
			})
		}
	}

	wg.Add(2)
	go stream(stdoutPipe, false)
	go stream(stderrPipe, true)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return false, exitErr.ExitCode()
		}
		return false, 1
	}
	return true, 0
}

// buildEnv merges the job's env (which includes resolved secrets) into the
// current process environment, with job env taking precedence.
func buildEnv(job *pb.JobDispatch) []string {
	seen := make(map[string]bool)
	var env []string
	for _, e := range job.Env {
		env = append(env, e.Key+"="+e.Value)
		seen[e.Key] = true
	}
	if job.Spec != nil {
		for _, e := range job.Spec.Env {
			if !seen[e.Key] {
				env = append(env, e.Key+"="+e.Value)
				seen[e.Key] = true
			}
		}
	}
	return env
}

func errChunk(job *pb.JobDispatch, msg string) *pb.LogChunk {
	return &pb.LogChunk{
		JobId:     job.JobId,
		RunId:     job.RunId,
		Timestamp: time.Now().UnixMilli(),
		Line:      "brazier: " + msg,
		Stderr:    true,
	}
}

// SplitCommand splits a shell command string into tokens (for display/logging).
func SplitCommand(s string) []string {
	return strings.Fields(s)
}
