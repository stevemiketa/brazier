import { describe, it, expect } from "@jest/globals";
import {
  pipeline, useWorkflow, job, stage, run,
  onBranch, onTag, onEvent,
} from "../pipeline";
import { encodePipelineSpec } from "../proto";

function buildPipeline() {
  return pipeline(
    useWorkflow("build-test-deploy", "v1.2.0"),
    job("lint", { commands: ["eslint .", "tsc --noEmit"] }),
    stage("test",
      job("unit", { commands: ["jest"] }),
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
}

describe("pipeline", () => {
  it("sets workflow ref", () => {
    const p = buildPipeline();
    expect(p.workflow.name).toBe("build-test-deploy");
    expect(p.workflow.version).toBe("v1.2.0");
  });

  it("has 4 top-level nodes", () => {
    expect(buildPipeline().nodes).toHaveLength(4);
  });

  it("build depends on test", () => {
    const p = buildPipeline();
    const build = p.nodes.find((n) => n.id === "build")!;
    expect(build.dependsOn).toEqual(["test"]);
  });

  it("test stage depends on lint", () => {
    const p = buildPipeline();
    const testNode = p.nodes.find((n) => n.id === "test")!;
    expect(testNode.dependsOn).toEqual(["lint"]);
    expect(testNode.stage?.jobs).toHaveLength(2);
  });

  it("deploy has branch condition", () => {
    const p = buildPipeline();
    const deploy = p.nodes.find((n) => n.id === "deploy")!;
    expect(deploy.conditions).toEqual(["branch == main"]);
  });

  it("job node has job spec", () => {
    const p = buildPipeline();
    const lint = p.nodes.find((n) => n.id === "lint")!;
    expect(lint.job?.commands).toEqual(["eslint .", "tsc --noEmit"]);
    expect(lint.stage).toBeUndefined();
  });

  it("stage node has no job spec", () => {
    const p = buildPipeline();
    const testNode = p.nodes.find((n) => n.id === "test")!;
    expect(testNode.job).toBeUndefined();
    expect(testNode.stage).toBeDefined();
  });

  it("artifact paths propagate", () => {
    const p = buildPipeline();
    const build = p.nodes.find((n) => n.id === "build")!;
    expect(build.job?.artifactPaths).toEqual(["dist/"]);
  });

  it("secrets propagate", () => {
    const p = buildPipeline();
    const deploy = p.nodes.find((n) => n.id === "deploy")!;
    expect(deploy.job?.secrets).toEqual(["DEPLOY_TOKEN"]);
  });

  it("env vars are mapped", () => {
    const p = pipeline(
      useWorkflow("wf", "v1"),
      job("lint", { commands: ["eslint ."], env: { NODE_ENV: "ci" } }),
    );
    const lint = p.nodes[0];
    expect(lint.job?.env).toEqual([{ key: "NODE_ENV", value: "ci" }]);
  });
});

describe("condition helpers", () => {
  it("onBranch", () => expect(onBranch("main")).toBe("branch == main"));
  it("onTag",    () => expect(onTag("v*")).toBe("tag == v*"));
  it("onEvent",  () => expect(onEvent("push")).toBe("event == push"));
});

describe("serialisation", () => {
  it("produces non-empty bytes", () => {
    const bytes = encodePipelineSpec(buildPipeline());
    expect(bytes.length).toBeGreaterThan(0);
  });

  it("bytes start with workflow ref field (tag 0x0a = field 1, wire type 2)", () => {
    const bytes = encodePipelineSpec(buildPipeline());
    // First byte should be tag for field 1 (workflow), wire type 2 (length-delimited) = 0x0a
    expect(bytes[0]).toBe(0x0a);
  });
});
