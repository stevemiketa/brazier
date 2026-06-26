package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	pb "github.com/brazier/brazier/proto/gen"
	"google.golang.org/protobuf/proto"
)

// Loader clones and caches workflow DAGs from the configured workflows repository.
type Loader struct {
	repoURL string
	cacheDir string

	mu    sync.Mutex
	cache map[string]*pb.WorkflowDAG // keyed by "name@version"
}

// NewLoader returns a Loader that fetches workflows from repoURL and caches
// cloned repos under cacheDir.
func NewLoader(repoURL, cacheDir string) *Loader {
	return &Loader{
		repoURL:  repoURL,
		cacheDir: cacheDir,
		cache:    make(map[string]*pb.WorkflowDAG),
	}
}

// Load returns the WorkflowDAG for the given name and version.
// It clones the repo (if needed), checks out the version, and executes the
// workflow Go file to extract the DAG shape.
func (l *Loader) Load(ctx context.Context, name, version string) (*pb.WorkflowDAG, error) {
	key := name + "@" + version

	l.mu.Lock()
	if dag, ok := l.cache[key]; ok {
		l.mu.Unlock()
		return dag, nil
	}
	l.mu.Unlock()

	repoDir, err := l.ensureRepo(ctx, version)
	if err != nil {
		return nil, err
	}

	dag, err := l.execWorkflow(ctx, repoDir, name)
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	l.cache[key] = dag
	l.mu.Unlock()

	return dag, nil
}

// ensureRepo clones or fetches the workflows repo and checks out the given version
// (semver tag or git ref). Returns the path to the checked-out repo.
func (l *Loader) ensureRepo(ctx context.Context, version string) (string, error) {
	dest := filepath.Join(l.cacheDir, "workflows")

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if _, err := os.Stat(filepath.Join(dest, ".git")); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, "git", "clone", "--no-tags", l.repoURL, dest)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git clone: %w", err)
		}
	} else {
		cmd := exec.CommandContext(ctx, "git", "-C", dest, "fetch", "--tags", "origin")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git fetch: %w", err)
		}
	}

	cmd := exec.CommandContext(ctx, "git", "-C", dest, "checkout", version)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git checkout %s: %w", version, err)
	}
	return dest, nil
}

// execWorkflow runs the workflow Go file named <name>.go in repoDir and
// deserializes the WorkflowDAG from its stdout.
func (l *Loader) execWorkflow(ctx context.Context, repoDir, name string) (*pb.WorkflowDAG, error) {
	wfFile := filepath.Join(repoDir, name+".go")
	if _, err := os.Stat(wfFile); err != nil {
		return nil, fmt.Errorf("workflow file not found: %s", wfFile)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", "run", wfFile)
	cmd.Dir = repoDir
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exec workflow %s: %w", name, err)
	}

	dag := &pb.WorkflowDAG{}
	if err := proto.Unmarshal(stdout.Bytes(), dag); err != nil {
		return nil, fmt.Errorf("unmarshal workflow dag: %w", err)
	}
	return dag, nil
}
