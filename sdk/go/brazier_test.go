package brazier_test

import (
	"testing"

	brazier "github.com/brazier/sdk/go"
	pb "github.com/brazier/brazier/proto/gen"
	"google.golang.org/protobuf/proto"
)

// buildPipeline constructs a representative pipeline for use across tests.
func buildPipeline() *brazier.Pipeline {
	lint := brazier.Job("lint", brazier.JobSpec{
		Commands: []string{"go vet ./...", "golangci-lint run"},
	})

	unit := brazier.Job("unit", brazier.JobSpec{Commands: []string{"go test ./..."}})
	integration := brazier.Job("integration", brazier.JobSpec{
		Commands: []string{"go test -tags=integration ./..."},
	})
	testStage := brazier.Stage("test", unit, integration).DependsOn("lint")

	build := brazier.Job("build", brazier.JobSpec{
		Commands:      []string{"go build -o bin/app ./cmd/app"},
		ArtifactPaths: []string{"bin/app"},
	}).DependsOn("test")

	deploy := brazier.Job("deploy", brazier.JobSpec{
		Commands: []string{"./scripts/deploy.sh"},
		Secrets:  []string{"DEPLOY_TOKEN"},
	}).DependsOn("build").When(brazier.OnBranch("main"))

	opts := []brazier.PipelineOption{brazier.UseWorkflow("build-test-deploy", "v1.2.0")}
	opts = append(opts, brazier.Nodes(lint, testStage, build, deploy)...)
	return brazier.NewPipeline(opts...)
}

func TestPipelineWorkflowRef(t *testing.T) {
	p := buildPipeline()
	spec := p.Spec()
	if spec.Workflow.Name != "build-test-deploy" {
		t.Errorf("workflow name = %q, want %q", spec.Workflow.Name, "build-test-deploy")
	}
	if spec.Workflow.Version != "v1.2.0" {
		t.Errorf("workflow version = %q, want %q", spec.Workflow.Version, "v1.2.0")
	}
}

func TestPipelineNodeCount(t *testing.T) {
	spec := buildPipeline().Spec()
	// lint, test (stage), build, deploy
	if len(spec.Nodes) != 4 {
		t.Errorf("node count = %d, want 4", len(spec.Nodes))
	}
}

func TestJobNodeDependsOn(t *testing.T) {
	spec := buildPipeline().Spec()
	// build depends on test
	var build *pb.Node
	for _, n := range spec.Nodes {
		if n.Id == "build" {
			build = n
		}
	}
	if build == nil {
		t.Fatal("build node not found")
	}
	if len(build.DependsOn) != 1 || build.DependsOn[0] != "test" {
		t.Errorf("build.DependsOn = %v, want [test]", build.DependsOn)
	}
}

func TestStageDependsOn(t *testing.T) {
	spec := buildPipeline().Spec()
	var testStage *pb.Node
	for _, n := range spec.Nodes {
		if n.Id == "test" {
			testStage = n
		}
	}
	if testStage == nil {
		t.Fatal("test stage not found")
	}
	if len(testStage.DependsOn) != 1 || testStage.DependsOn[0] != "lint" {
		t.Errorf("test.DependsOn = %v, want [lint]", testStage.DependsOn)
	}
	s := testStage.GetStage()
	if s == nil {
		t.Fatal("test node is not a stage")
	}
	if len(s.Jobs) != 2 {
		t.Errorf("stage job count = %d, want 2", len(s.Jobs))
	}
}

func TestJobCondition(t *testing.T) {
	spec := buildPipeline().Spec()
	var deploy *pb.Node
	for _, n := range spec.Nodes {
		if n.Id == "deploy" {
			deploy = n
		}
	}
	if deploy == nil {
		t.Fatal("deploy node not found")
	}
	if len(deploy.Conditions) != 1 || deploy.Conditions[0] != "branch == main" {
		t.Errorf("deploy.Conditions = %v, want [branch == main]", deploy.Conditions)
	}
}

func TestConditionHelpers(t *testing.T) {
	cases := []struct {
		cond brazier.Condition
		want string
	}{
		{brazier.OnBranch("main"), "branch == main"},
		{brazier.OnTag("v*"), "tag == v*"},
		{brazier.OnEvent("push"), "event == push"},
	}
	for _, c := range cases {
		if string(c.cond) != c.want {
			t.Errorf("condition = %q, want %q", c.cond, c.want)
		}
	}
}

func TestSerializationRoundTrip(t *testing.T) {
	p := buildPipeline()
	b, err := p.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := &pb.PipelineSpec{}
	if err := proto.Unmarshal(b, got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !proto.Equal(p.Spec(), got) {
		t.Errorf("round-trip mismatch")
	}
}

func TestWorkflowDAG(t *testing.T) {
	wf := brazier.NewWorkflow("build-test-deploy", "v1.2.0",
		brazier.WorkflowNodes(
			brazier.Node("lint"),
			brazier.Node("test").DependsOn("lint"),
			brazier.Node("build").DependsOn("test"),
			brazier.Node("deploy").DependsOn("build").When(brazier.OnBranch("main")),
		),
	)
	dag := wf.DAG()
	if dag.Name != "build-test-deploy" {
		t.Errorf("name = %q", dag.Name)
	}
	if len(dag.Nodes) != 4 {
		t.Errorf("node count = %d, want 4", len(dag.Nodes))
	}
	// verify deploy has condition
	deploy := dag.Nodes[3]
	if len(deploy.Conditions) != 1 {
		t.Errorf("deploy conditions = %v", deploy.Conditions)
	}
}

func TestEnvVars(t *testing.T) {
	p := brazier.NewPipeline(
		brazier.UseWorkflow("wf", "v1"),
	)
	opts := brazier.Nodes(brazier.Job("lint", brazier.JobSpec{
		Commands: []string{"go vet ./..."},
		Env:      map[string]string{"GOFLAGS": "-mod=vendor"},
	}))
	for _, o := range opts {
		_ = o // apply via NewPipeline
	}
	p2 := brazier.NewPipeline(append(
		[]brazier.PipelineOption{brazier.UseWorkflow("wf", "v1")},
		brazier.Nodes(brazier.Job("lint", brazier.JobSpec{
			Commands: []string{"go vet ./..."},
			Env:      map[string]string{"GOFLAGS": "-mod=vendor"},
		}))...,
	)...)
	spec := p2.Spec()
	job := spec.Nodes[0].GetJob()
	if len(job.Env) != 1 || job.Env[0].Key != "GOFLAGS" {
		t.Errorf("env = %v", job.Env)
	}
	_ = p
}
