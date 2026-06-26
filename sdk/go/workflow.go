package brazier

import (
	"os"

	pb "github.com/brazier/brazier/proto/gen"
	"google.golang.org/protobuf/proto"
)

// Workflow is the in-memory DAG definition used by workflow repo files.
type Workflow struct {
	dag *pb.WorkflowDAG
}

// WorkflowOption is a functional option applied to a Workflow.
type WorkflowOption func(*Workflow)

// NewWorkflow constructs a named, versioned workflow DAG.
func NewWorkflow(name, version string, opts ...WorkflowOption) *Workflow {
	w := &Workflow{dag: &pb.WorkflowDAG{Name: name, Version: version}}
	for _, o := range opts {
		o(w)
	}
	return w
}

// DAG returns the underlying WorkflowDAG (useful for testing).
func (w *Workflow) DAG() *pb.WorkflowDAG { return w.dag }

// RunWorkflow serializes w to protobuf, writes it to stdout, and exits 0.
// It is the last call in any workflow definition file's main().
func RunWorkflow(w *Workflow) {
	b, err := proto.Marshal(w.dag)
	if err != nil {
		_, _ = os.Stderr.Write([]byte("brazier: marshal workflow: " + err.Error() + "\n"))
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(b); err != nil {
		_, _ = os.Stderr.Write([]byte("brazier: write stdout: " + err.Error() + "\n"))
		os.Exit(1)
	}
	os.Exit(0)
}

// -------------------------------------------------------------------
// Workflow node builder
// -------------------------------------------------------------------

// WorkflowNode is a DAG node in a workflow definition (shape only, no job config).
type WorkflowNode struct {
	node *pb.Node
}

// Node creates a workflow DAG node with the given id.
func Node(id string) *WorkflowNode {
	return &WorkflowNode{node: &pb.Node{Id: id}}
}

// DependsOn declares that this node depends on the named nodes.
func (n *WorkflowNode) DependsOn(ids ...string) *WorkflowNode {
	n.node.DependsOn = append(n.node.DependsOn, ids...)
	return n
}

// When attaches condition expressions to the node.
func (n *WorkflowNode) When(conditions ...Condition) *WorkflowNode {
	for _, c := range conditions {
		n.node.Conditions = append(n.node.Conditions, string(c))
	}
	return n
}

// WorkflowNodes converts WorkflowNode values into WorkflowOptions for NewWorkflow.
func WorkflowNodes(ns ...*WorkflowNode) WorkflowOption {
	return func(w *Workflow) {
		for _, n := range ns {
			w.dag.Nodes = append(w.dag.Nodes, n.node)
		}
	}
}
