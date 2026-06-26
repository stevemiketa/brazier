//go:build ignore

package main

import brazier "github.com/brazier/sdk/go"

func main() {
	p := brazier.NewPipeline(append(
		[]brazier.PipelineOption{
			brazier.UseWorkflow("build-test-deploy", "v1.2.0"),
		},
		brazier.Nodes(
			brazier.Job("lint", brazier.JobSpec{
				Commands: []string{"go vet ./...", "golangci-lint run"},
			}),

			brazier.Stage("test",
				brazier.Job("unit", brazier.JobSpec{
					Commands: []string{"go test ./..."},
				}),
				brazier.Job("integration", brazier.JobSpec{
					Commands: []string{"go test -tags=integration ./..."},
				}),
			).DependsOn("lint"),

			brazier.Job("build", brazier.JobSpec{
				Commands:      []string{"go build -o bin/app ./cmd/app"},
				ArtifactPaths: []string{"bin/app"},
			}).DependsOn("test"),

			brazier.Job("deploy", brazier.JobSpec{
				Commands: []string{"./scripts/deploy.sh"},
				Secrets:  []string{"DEPLOY_TOKEN"},
			}).DependsOn("build").When(brazier.OnBranch("main")),
		)...,
	)...)

	brazier.Run(p)
}
