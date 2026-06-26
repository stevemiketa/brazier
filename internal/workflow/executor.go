// Package workflow handles loading and executing Brazierfiles and workflow DAG files.
package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	pb "github.com/brazier/brazier/proto/gen"
	"google.golang.org/protobuf/proto"
)

// ExecBrazierfile detects and executes the Brazierfile in dir, returning the
// deserialized PipelineSpec from its stdout.
func ExecBrazierfile(ctx context.Context, dir string) (*pb.PipelineSpec, error) {
	path, lang, err := detectBrazierfile(dir)
	if err != nil {
		return nil, err
	}

	var stdout bytes.Buffer
	var cmd *exec.Cmd

	switch lang {
	case "go":
		cmd = exec.CommandContext(ctx, "go", "run", path)
	case "ts":
		cmd = exec.CommandContext(ctx, "npx", "ts-node", path)
	default:
		return nil, fmt.Errorf("unsupported brazierfile language: %s", lang)
	}

	cmd.Dir = dir
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exec brazierfile: %w", err)
	}

	spec := &pb.PipelineSpec{}
	if err := proto.Unmarshal(stdout.Bytes(), spec); err != nil {
		return nil, fmt.Errorf("unmarshal pipeline spec: %w", err)
	}
	return spec, nil
}

func detectBrazierfile(dir string) (path, lang string, err error) {
	candidates := []struct {
		name string
		lang string
	}{
		{"Brazierfile.go", "go"},
		{"Brazierfile.ts", "ts"},
	}
	for _, c := range candidates {
		p := filepath.Join(dir, c.name)
		if _, err := os.Stat(p); err == nil {
			return p, c.lang, nil
		}
	}
	return "", "", fmt.Errorf("no Brazierfile.go or Brazierfile.ts found in %s", dir)
}
