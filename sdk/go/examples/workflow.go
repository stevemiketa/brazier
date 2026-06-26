//go:build ignore

package main

import brazier "github.com/brazier/sdk/go"

func main() {
	wf := brazier.NewWorkflow("build-test-deploy", "v1.2.0",
		brazier.WorkflowNodes(
			brazier.Node("lint"),
			brazier.Node("test").DependsOn("lint"),
			brazier.Node("build").DependsOn("test"),
			brazier.Node("deploy").DependsOn("build").When(brazier.OnBranch("main")),
		),
	)

	brazier.RunWorkflow(wf)
}
