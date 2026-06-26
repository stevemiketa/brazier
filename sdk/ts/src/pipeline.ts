import * as process from "process";
import { encodePipelineSpec, PipelineSpec, Node, JobSpec, EnvVar } from "./proto";

// ---------------------------------------------------------------------------
// Condition primitives
// ---------------------------------------------------------------------------

export type Condition = string;

export const onBranch = (name: string): Condition => `branch == ${name}`;
export const onTag = (pattern: string): Condition => `tag == ${pattern}`;
export const onEvent = (event: string): Condition => `event == ${event}`;

// ---------------------------------------------------------------------------
// Job builder
// ---------------------------------------------------------------------------

export interface JobOptions {
  commands: string[];
  env?: Record<string, string>;
  secrets?: string[];
  artifactPaths?: string[];
}

export class JobNode {
  readonly _node: Node;

  constructor(id: string, opts: JobOptions) {
    const env: EnvVar[] = Object.entries(opts.env ?? {}).map(([key, value]) => ({ key, value }));
    this._node = {
      id,
      dependsOn: [],
      conditions: [],
      job: {
        commands: opts.commands,
        env,
        secrets: opts.secrets ?? [],
        artifactPaths: opts.artifactPaths ?? [],
      },
    };
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

export function job(id: string, opts: JobOptions): JobNode {
  return new JobNode(id, opts);
}

// ---------------------------------------------------------------------------
// Stage builder
// ---------------------------------------------------------------------------

export class StageNode {
  readonly _node: Node;

  constructor(id: string, jobs: JobNode[]) {
    this._node = {
      id,
      dependsOn: [],
      conditions: [],
      stage: { jobs: jobs.map((j) => j._node) },
    };
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

export function stage(id: string, ...jobs: JobNode[]): StageNode {
  return new StageNode(id, jobs);
}

// ---------------------------------------------------------------------------
// Pipeline builder
// ---------------------------------------------------------------------------

export type PipelineOption = (spec: PipelineSpec) => void;

export function useWorkflow(name: string, version: string): PipelineOption {
  return (spec) => {
    spec.workflow = { name, version };
  };
}

export function pipeline(...opts: (PipelineOption | JobNode | StageNode)[]): PipelineSpec {
  const spec: PipelineSpec = {
    workflow: { name: "", version: "" },
    nodes: [],
  };
  for (const opt of opts) {
    if (opt instanceof JobNode || opt instanceof StageNode) {
      spec.nodes.push(opt._node);
    } else {
      opt(spec);
    }
  }
  return spec;
}

/**
 * Serialise spec to protobuf bytes, write to stdout, and exit 0.
 * This is the last call in any Brazierfile.ts main().
 */
export function run(spec: PipelineSpec): never {
  const bytes = encodePipelineSpec(spec);
  process.stdout.write(bytes);
  process.exit(0);
}
