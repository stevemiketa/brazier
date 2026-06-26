// Package brazier is the Go SDK for defining Brazier pipelines and workflows.
// A Brazierfile calls NewPipeline(...).then Run(p) to serialize a PipelineSpec
// to stdout, which the master service reads and deserializes.
package brazier

import (
	"os"

	pb "github.com/brazier/brazier/proto/gen"
	"google.golang.org/protobuf/proto"
)

// Pipeline is the in-memory representation built by NewPipeline.
type Pipeline struct {
	spec *pb.PipelineSpec
}

// PipelineOption is a functional option applied to a Pipeline.
type PipelineOption func(*Pipeline)

// NewPipeline constructs a Pipeline from the given options.
func NewPipeline(opts ...PipelineOption) *Pipeline {
	p := &Pipeline{spec: &pb.PipelineSpec{}}
	for _, o := range opts {
		o(p)
	}
	return p
}

// UseWorkflow sets the workflow DAG reference.
func UseWorkflow(name, version string) PipelineOption {
	return func(p *Pipeline) {
		p.spec.Workflow = &pb.WorkflowRef{Name: name, Version: version}
	}
}

// Spec returns the underlying PipelineSpec (useful for testing).
func (p *Pipeline) Spec() *pb.PipelineSpec { return p.spec }

// Marshal serializes p to protobuf bytes.
func (p *Pipeline) Marshal() ([]byte, error) { return proto.Marshal(p.spec) }

// Run serializes p to protobuf, writes it to stdout, and exits 0.
// It is the last call in any Brazierfile main().
func Run(p *Pipeline) {
	b, err := proto.Marshal(p.spec)
	if err != nil {
		_, _ = os.Stderr.Write([]byte("brazier: marshal: " + err.Error() + "\n"))
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(b); err != nil {
		_, _ = os.Stderr.Write([]byte("brazier: write stdout: " + err.Error() + "\n"))
		os.Exit(1)
	}
	os.Exit(0)
}

// -------------------------------------------------------------------
// Job builder
// -------------------------------------------------------------------

// JobSpec holds the configuration for a single job node.
type JobSpec struct {
	Commands      []string
	Env           map[string]string
	Secrets       []string
	ArtifactPaths []string
}

// JobNode is a node builder that supports method chaining.
type JobNode struct {
	node *pb.Node
}

// Job creates a job node option.
func Job(id string, spec JobSpec) *JobNode {
	env := make([]*pb.EnvVar, 0, len(spec.Env))
	for k, v := range spec.Env {
		env = append(env, &pb.EnvVar{Key: k, Value: v})
	}
	return &JobNode{node: &pb.Node{
		Id: id,
		Kind: &pb.Node_Job{Job: &pb.JobSpec{
			Commands:      spec.Commands,
			Env:           env,
			Secrets:       spec.Secrets,
			ArtifactPaths: spec.ArtifactPaths,
		}},
	}}
}

// DependsOn declares that this job must run after the named nodes.
func (j *JobNode) DependsOn(ids ...string) *JobNode {
	j.node.DependsOn = append(j.node.DependsOn, ids...)
	return j
}

// When attaches a condition expression evaluated at scheduling time.
func (j *JobNode) When(conditions ...Condition) *JobNode {
	for _, c := range conditions {
		j.node.Conditions = append(j.node.Conditions, string(c))
	}
	return j
}

// option makes JobNode usable as a PipelineOption.
func (j *JobNode) option() PipelineOption {
	return func(p *Pipeline) {
		p.spec.Nodes = append(p.spec.Nodes, j.node)
	}
}

// -------------------------------------------------------------------
// Stage builder
// -------------------------------------------------------------------

// StageNode is a node builder for stages (sets of parallel jobs).
type StageNode struct {
	node *pb.Node
}

// Stage creates a stage node that runs the given jobs in parallel.
func Stage(id string, jobs ...*JobNode) *StageNode {
	inner := make([]*pb.Node, len(jobs))
	for i, j := range jobs {
		inner[i] = j.node
	}
	return &StageNode{node: &pb.Node{
		Id: id,
		Kind: &pb.Node_Stage{Stage: &pb.StageSpec{Jobs: inner}},
	}}
}

// DependsOn declares that this stage must run after the named nodes.
func (s *StageNode) DependsOn(ids ...string) *StageNode {
	s.node.DependsOn = append(s.node.DependsOn, ids...)
	return s
}

// When attaches condition expressions to the stage node.
func (s *StageNode) When(conditions ...Condition) *StageNode {
	for _, c := range conditions {
		s.node.Conditions = append(s.node.Conditions, string(c))
	}
	return s
}

// option makes StageNode usable as a PipelineOption.
func (s *StageNode) option() PipelineOption {
	return func(p *Pipeline) {
		p.spec.Nodes = append(p.spec.Nodes, s.node)
	}
}

// -------------------------------------------------------------------
// Node-as-option helpers so callers can pass JobNode/StageNode
// directly to NewPipeline without calling .option() explicitly.
// -------------------------------------------------------------------

// Nodeable is implemented by JobNode and StageNode.
type Nodeable interface {
	option() PipelineOption
}

// NewPipeline accepts PipelineOption values, but JobNode and StageNode
// are not directly PipelineOption. We expose a helper so callers can mix
// UseWorkflow(...) with node builders ergonomically.
//
// Usage:
//
//	NewPipeline(
//	    UseWorkflow("wf", "v1"),
//	    Nodes(
//	        Job("lint", ...),
//	        Stage("test", ...),
//	    )...,
//	)

// Nodes converts Nodeable values to PipelineOptions for use in NewPipeline.
func Nodes(ns ...Nodeable) []PipelineOption {
	opts := make([]PipelineOption, len(ns))
	for i, n := range ns {
		opts[i] = n.option()
	}
	return opts
}

// -------------------------------------------------------------------
// Conditions
// -------------------------------------------------------------------

// Condition is an opaque expression string evaluated by the master scheduler.
type Condition string

// OnBranch returns a condition that is true when the triggering branch matches name.
func OnBranch(name string) Condition { return Condition("branch == " + name) }

// OnTag returns a condition that is true when the triggering ref is a tag matching pattern.
func OnTag(pattern string) Condition { return Condition("tag == " + pattern) }

// OnEvent returns a condition that is true when the triggering GitHub event matches.
func OnEvent(event string) Condition { return Condition("event == " + event) }
