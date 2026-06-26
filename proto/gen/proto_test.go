package brazierv1_test

import (
	"testing"

	brazierv1 "github.com/brazier/brazier/proto/gen"
	"google.golang.org/protobuf/proto"
)

func TestPipelineSpecRoundTrip(t *testing.T) {
	orig := &brazierv1.PipelineSpec{
		Workflow: &brazierv1.WorkflowRef{
			Name:    "build-test-deploy",
			Version: "v1.2.0",
		},
		Nodes: []*brazierv1.Node{
			{
				Id: "lint",
				Kind: &brazierv1.Node_Job{
					Job: &brazierv1.JobSpec{
						Commands: []string{"go vet ./...", "golangci-lint run"},
					},
				},
			},
			{
				Id:        "build",
				DependsOn: []string{"lint"},
				Kind: &brazierv1.Node_Job{
					Job: &brazierv1.JobSpec{
						Commands:      []string{"go build -o bin/app ./cmd/app"},
						ArtifactPaths: []string{"bin/app"},
					},
				},
			},
			{
				Id:         "deploy",
				DependsOn:  []string{"build"},
				Conditions: []string{"branch == main"},
				Kind: &brazierv1.Node_Job{
					Job: &brazierv1.JobSpec{
						Commands: []string{"./scripts/deploy.sh"},
						Secrets:  []string{"DEPLOY_TOKEN"},
					},
				},
			},
		},
	}

	b, err := proto.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got := &brazierv1.PipelineSpec{}
	if err := proto.Unmarshal(b, got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !proto.Equal(orig, got) {
		t.Errorf("round-trip mismatch:\n  orig: %v\n   got: %v", orig, got)
	}
}

func TestJobDispatchRoundTrip(t *testing.T) {
	orig := &brazierv1.JobDispatch{
		JobId: "job-123",
		RunId: "run-456",
		Spec: &brazierv1.JobSpec{
			Commands: []string{"echo hello"},
			Env:      []*brazierv1.EnvVar{{Key: "CI", Value: "true"}},
		},
		Env: []*brazierv1.EnvVar{
			{Key: "DEPLOY_TOKEN", Value: "secret-value"},
		},
	}

	b, err := proto.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got := &brazierv1.JobDispatch{}
	if err := proto.Unmarshal(b, got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !proto.Equal(orig, got) {
		t.Errorf("round-trip mismatch:\n  orig: %v\n   got: %v", orig, got)
	}
}

func TestStageSpecRoundTrip(t *testing.T) {
	orig := &brazierv1.PipelineSpec{
		Workflow: &brazierv1.WorkflowRef{Name: "wf", Version: "v1.0.0"},
		Nodes: []*brazierv1.Node{
			{
				Id: "test",
				Kind: &brazierv1.Node_Stage{
					Stage: &brazierv1.StageSpec{
						Jobs: []*brazierv1.Node{
							{
								Id: "unit",
								Kind: &brazierv1.Node_Job{
									Job: &brazierv1.JobSpec{Commands: []string{"go test ./..."}},
								},
							},
							{
								Id: "integration",
								Kind: &brazierv1.Node_Job{
									Job: &brazierv1.JobSpec{Commands: []string{"go test -tags=integration ./..."}},
								},
							},
						},
					},
				},
			},
		},
	}

	b, err := proto.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got := &brazierv1.PipelineSpec{}
	if err := proto.Unmarshal(b, got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !proto.Equal(orig, got) {
		t.Errorf("round-trip mismatch:\n  orig: %v\n   got: %v", orig, got)
	}
}
