package registry_test

import (
	"testing"

	"github.com/brazier/brazier/internal/registry"
)

func TestNewRegistry(t *testing.T) {
	r := registry.New()
	agents := r.List()
	if len(agents) != 0 {
		t.Errorf("expected empty registry, got %d agents", len(agents))
	}
}

func TestJobDoneNoAgent(t *testing.T) {
	r := registry.New()
	// Should not panic when agent is unknown.
	r.JobDone("nonexistent-agent")
}
