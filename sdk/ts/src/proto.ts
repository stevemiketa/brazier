/**
 * Hand-written TypeScript types mirroring proto/pipeline.proto.
 * Serialization uses @bufbuild/protobuf's BinaryWriter to produce
 * wire-compatible protobuf bytes.
 */

import { BinaryWriter } from "@bufbuild/protobuf/wire";

export interface EnvVar {
  key: string;
  value: string;
}

export interface JobSpec {
  commands: string[];
  env: EnvVar[];
  secrets: string[];
  artifactPaths: string[];
}

export interface StageSpec {
  jobs: Node[];
}

export interface Node {
  id: string;
  dependsOn: string[];
  conditions: string[];
  job?: JobSpec;
  stage?: StageSpec;
}

export interface WorkflowRef {
  name: string;
  version: string;
}

export interface PipelineSpec {
  workflow: WorkflowRef;
  nodes: Node[];
}

export interface WorkflowDAG {
  name: string;
  version: string;
  nodes: Node[];
}

// ---------------------------------------------------------------------------
// Protobuf binary serialisation
// Each function encodes the corresponding message using field numbers from
// the .proto definitions.
// ---------------------------------------------------------------------------

function writeEnvVar(w: BinaryWriter, env: EnvVar): void {
  if (env.key) w.tag(1, 2).string(env.key);
  if (env.value) w.tag(2, 2).string(env.value);
}

function writeJobSpec(w: BinaryWriter, spec: JobSpec): void {
  for (const cmd of spec.commands) w.tag(1, 2).string(cmd);
  for (const env of spec.env) {
    const nested = new BinaryWriter();
    writeEnvVar(nested, env);
    w.tag(2, 2).bytes(nested.finish());
  }
  for (const s of spec.secrets) w.tag(3, 2).string(s);
  for (const p of spec.artifactPaths) w.tag(4, 2).string(p);
}

function writeNode(w: BinaryWriter, node: Node): void {
  if (node.id) w.tag(1, 2).string(node.id);
  for (const dep of node.dependsOn) w.tag(2, 2).string(dep);
  for (const cond of node.conditions) w.tag(3, 2).string(cond);
  if (node.job !== undefined) {
    const nested = new BinaryWriter();
    writeJobSpec(nested, node.job);
    w.tag(4, 2).bytes(nested.finish());
  } else if (node.stage !== undefined) {
    const nested = new BinaryWriter();
    writeStageSpec(nested, node.stage);
    w.tag(5, 2).bytes(nested.finish());
  }
}

function writeStageSpec(w: BinaryWriter, stage: StageSpec): void {
  for (const job of stage.jobs) {
    const nested = new BinaryWriter();
    writeNode(nested, job);
    w.tag(1, 2).bytes(nested.finish());
  }
}

function writeWorkflowRef(w: BinaryWriter, ref: WorkflowRef): void {
  if (ref.name) w.tag(1, 2).string(ref.name);
  if (ref.version) w.tag(2, 2).string(ref.version);
}

/** Serialise a PipelineSpec to protobuf binary bytes. */
export function encodePipelineSpec(spec: PipelineSpec): Uint8Array {
  const w = new BinaryWriter();
  const wfNested = new BinaryWriter();
  writeWorkflowRef(wfNested, spec.workflow);
  w.tag(1, 2).bytes(wfNested.finish());
  for (const node of spec.nodes) {
    const nested = new BinaryWriter();
    writeNode(nested, node);
    w.tag(2, 2).bytes(nested.finish());
  }
  return w.finish();
}

/** Serialise a WorkflowDAG to protobuf binary bytes. */
export function encodeWorkflowDAG(dag: WorkflowDAG): Uint8Array {
  const w = new BinaryWriter();
  if (dag.name) w.tag(1, 2).string(dag.name);
  if (dag.version) w.tag(2, 2).string(dag.version);
  for (const node of dag.nodes) {
    const nested = new BinaryWriter();
    writeNode(nested, node);
    w.tag(3, 2).bytes(nested.finish());
  }
  return w.finish();
}
