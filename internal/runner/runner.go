// Package runner defines the Runner interface and its implementations.
package runner

import (
	"context"

	pb "github.com/brazier/brazier/proto/gen"
)

// Runner dispatches and cancels jobs on a particular execution backend.
type Runner interface {
	Dispatch(ctx context.Context, job *pb.JobDispatch) error
	Cancel(ctx context.Context, jobID string) error
}
