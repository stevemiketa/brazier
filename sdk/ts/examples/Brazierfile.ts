import { pipeline, useWorkflow, job, stage, run, onBranch } from "../src";

const p = pipeline(
  useWorkflow("build-test-deploy", "v1.2.0"),

  job("lint", { commands: ["eslint .", "tsc --noEmit"] }),

  stage("test",
    job("unit", { commands: ["jest --testPathPattern=unit"] }),
    job("e2e",  { commands: ["playwright test"] }),
  ).dependsOn("lint"),

  job("build", {
    commands: ["npm run build"],
    artifactPaths: ["dist/"],
  }).dependsOn("test"),

  job("deploy", {
    commands: ["./scripts/deploy.sh"],
    secrets: ["DEPLOY_TOKEN"],
  }).dependsOn("build").when(onBranch("main")),
);

run(p);
