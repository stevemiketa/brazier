package runner

import (
	"context"
	"fmt"

	pb "github.com/brazier/brazier/proto/gen"
)

// NomadRunner is a stub Runner that submits jobs to HashiCorp Nomad.
type NomadRunner struct {
	addr string // Nomad HTTP API address
}

// NewNomadRunner returns a NomadRunner pointed at addr (e.g. "http://localhost:4646").
func NewNomadRunner(addr string) *NomadRunner {
	return &NomadRunner{addr: addr}
}

func (n *NomadRunner) Dispatch(_ context.Context, job *pb.JobDispatch) error {
	return fmt.Errorf("nomad runner: not implemented (job %s)", job.JobId)
}

func (n *NomadRunner) Cancel(_ context.Context, jobID string) error {
	return fmt.Errorf("nomad runner: not implemented (job %s)", jobID)
}
