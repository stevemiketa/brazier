/**
 * Hand-written service descriptor mirroring api.proto.
 * Compatible with @connectrpc/connect v2 + @bufbuild/protobuf v2.
 * MethodKind and ServiceType were removed in v2; methodKind is now a string literal.
 */

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

export const BrazierAPI = {
  typeName: "brazier.BrazierAPI",
  methods: {
    listRuns: {
      name: "ListRuns",
      I: {} as ListRunsRequest,
      O: {} as RunList,
      kind: "unary" as const,
    },
    getRun: {
      name: "GetRun",
      I: {} as RunID,
      O: {} as RunStatus,
      kind: "unary" as const,
    },
    cancelRun: {
      name: "CancelRun",
      I: {} as RunID,
      O: {} as Empty,
      kind: "unary" as const,
    },
    streamLogs: {
      name: "StreamLogs",
      I: {} as RunID,
      O: {} as LogChunk,
      kind: "server_streaming" as const,
    },
    listAgents: {
      name: "ListAgents",
      I: {} as Empty,
      O: {} as AgentList,
      kind: "unary" as const,
    },
    listWorkflows: {
      name: "ListWorkflows",
      I: {} as Empty,
      O: {} as WorkflowList,
      kind: "unary" as const,
    },
    getWorkflow: {
      name: "GetWorkflow",
      I: {} as WorkflowRef,
      O: {} as WorkflowDAG,
      kind: "unary" as const,
    },
  },
};
