package runner

import (
	"context"
	"fmt"

	pb "github.com/brazier/brazier/proto/gen"
)

// K8sRunner is a stub Runner that creates Kubernetes Job resources.
type K8sRunner struct {
	namespace  string
	image      string // container image used for all jobs
}

// NewK8sRunner returns a K8sRunner targeting the given namespace.
func NewK8sRunner(namespace, image string) *K8sRunner {
	return &K8sRunner{namespace: namespace, image: image}
}

func (k *K8sRunner) Dispatch(_ context.Context, job *pb.JobDispatch) error {
	return fmt.Errorf("k8s runner: not implemented (job %s)", job.JobId)
}

func (k *K8sRunner) Cancel(_ context.Context, jobID string) error {
	return fmt.Errorf("k8s runner: not implemented (job %s)", jobID)
}
