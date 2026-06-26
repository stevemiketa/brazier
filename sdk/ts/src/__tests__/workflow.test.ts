import { describe, it, expect } from "@jest/globals";
import { newWorkflow, node } from "../workflow";
import { onBranch } from "../pipeline";
import { encodeWorkflowDAG } from "../proto";

describe("workflow", () => {
  it("sets name and version", () => {
    const wf = newWorkflow("my-wf", "v2.0.0");
    expect(wf.name).toBe("my-wf");
    expect(wf.version).toBe("v2.0.0");
  });

  it("has 4 nodes", () => {
    const wf = newWorkflow("wf", "v1",
      node("lint"),
      node("test").dependsOn("lint"),
      node("build").dependsOn("test"),
      node("deploy").dependsOn("build").when(onBranch("main")),
    );
    expect(wf.nodes).toHaveLength(4);
  });

  it("preserves dependency chain", () => {
    const wf = newWorkflow("wf", "v1",
      node("a"),
      node("b").dependsOn("a"),
      node("c").dependsOn("b"),
    );
    expect(wf.nodes[1].dependsOn).toEqual(["a"]);
    expect(wf.nodes[2].dependsOn).toEqual(["b"]);
  });

  it("conditions attach to nodes", () => {
    const wf = newWorkflow("wf", "v1",
      node("deploy").when(onBranch("main")),
    );
    expect(wf.nodes[0].conditions).toEqual(["branch == main"]);
  });

  it("serialises to bytes", () => {
    const wf = newWorkflow("wf", "v1", node("lint"));
    const bytes = encodeWorkflowDAG(wf);
    expect(bytes.length).toBeGreaterThan(0);
  });
});
