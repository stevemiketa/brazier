/**
 * Hand-written connect-es service descriptors mirroring api.proto.
 * These are used to type the gRPC-web client without running protoc.
 */
import { ServiceType } from "@connectrpc/connect";
import { MethodKind } from "@bufbuild/protobuf";

// ---------- Message shapes ----------

export interface RunID { id: string }
export interface Empty {}
export interface RunStatus { runId: string; state: string; nodes: string[] }
export interface ListRunsRequest { project: string; limit: number }
export interface RunList { runs: RunStatus[] }
export interface AgentInfo { agentId: string; name: string; labels: string[]; capacity: number; active: number }
export interface AgentList { agents: AgentInfo[] }
export interface WorkflowList { names: string[] }
export interface WorkflowRef { name: string; version: string }
export interface WorkflowNode { id: string; dependsOn: string[]; conditions: string[] }
export interface WorkflowDAG { name: string; version: string; nodes: WorkflowNode[] }
export interface LogChunk { jobId: string; runId: string; timestamp: bigint; line: string; stderr: boolean }

// ---------- Service descriptor ----------
// We use a minimal duck-typed ServiceType compatible with connectrpc.

export const BrazierAPI = {
  typeName: "brazier.BrazierAPI",
  methods: {
    listRuns: {
      name: "ListRuns",
      I: {} as ListRunsRequest,
      O: {} as RunList,
      kind: MethodKind.Unary,
    },
    getRun: {
      name: "GetRun",
      I: {} as RunID,
      O: {} as RunStatus,
      kind: MethodKind.Unary,
    },
    cancelRun: {
      name: "CancelRun",
      I: {} as RunID,
      O: {} as Empty,
      kind: MethodKind.Unary,
    },
    streamLogs: {
      name: "StreamLogs",
      I: {} as RunID,
      O: {} as LogChunk,
      kind: MethodKind.ServerStreaming,
    },
    listAgents: {
      name: "ListAgents",
      I: {} as Empty,
      O: {} as AgentList,
      kind: MethodKind.Unary,
    },
    listWorkflows: {
      name: "ListWorkflows",
      I: {} as Empty,
      O: {} as WorkflowList,
      kind: MethodKind.Unary,
    },
    getWorkflow: {
      name: "GetWorkflow",
      I: {} as WorkflowRef,
      O: {} as WorkflowDAG,
      kind: MethodKind.Unary,
    },
  },
} satisfies ServiceType;
