import { newWorkflow, node, runWorkflow, onBranch } from "../src";

const wf = newWorkflow("build-test-deploy", "v1.2.0",
  node("lint"),
  node("test").dependsOn("lint"),
  node("build").dependsOn("test"),
  node("deploy").dependsOn("build").when(onBranch("main")),
);

runWorkflow(wf);
