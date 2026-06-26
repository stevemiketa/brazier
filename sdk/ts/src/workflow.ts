import * as process from "process";
import { encodeWorkflowDAG, WorkflowDAG, Node } from "./proto";
import { Condition } from "./pipeline";

// ---------------------------------------------------------------------------
// Workflow node builder (shape only — no job config)
// ---------------------------------------------------------------------------

export class WorkflowNode {
  readonly _node: Node;

  constructor(id: string) {
    this._node = { id, dependsOn: [], conditions: [] };
  }

  dependsOn(...ids: string[]): this {
    this._node.dependsOn.push(...ids);
    return this;
  }

  when(...conditions: Condition[]): this {
    this._node.conditions.push(...conditions);
    return this;
  }
}

export function node(id: string): WorkflowNode {
  return new WorkflowNode(id);
}

// ---------------------------------------------------------------------------
// Workflow builder
// ---------------------------------------------------------------------------

export function newWorkflow(name: string, version: string, ...nodes: WorkflowNode[]): WorkflowDAG {
  return { name, version, nodes: nodes.map((n) => n._node) };
}

/**
 * Serialise dag to protobuf bytes, write to stdout, and exit 0.
 * This is the last call in any workflow definition file.
 */
export function runWorkflow(dag: WorkflowDAG): never {
  const bytes = encodeWorkflowDAG(dag);
  process.stdout.write(bytes);
  process.exit(0);
}
