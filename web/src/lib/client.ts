/**
 * Thin JSON client for the Brazier gRPC-web API using the Connect protocol.
 * Avoids generated descriptors by calling the Connect JSON HTTP endpoints directly.
 * POST /<package>.<Service>/<Method> with Content-Type: application/json
 */
import type {
  RunID, RunStatus, ListRunsRequest, RunList,
  AgentList, WorkflowList, WorkflowRef, WorkflowDAG, LogChunk, Empty,
} from "./brazier_connect";
import { getApiKey } from "./auth";

const BASE = "/brazier.BrazierAPI";

function authHeaders(): Record<string, string> {
  const key = getApiKey();
  return key ? { Authorization: `Bearer ${key}` } : {};
}

async function call<I, O>(method: string, body: I): Promise<O> {
  const res = await fetch(`${BASE}/${method}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${method} failed (${res.status}): ${text}`);
  }
  return res.json() as Promise<O>;
}

async function* stream<I, O>(method: string, body: I): AsyncGenerator<O> {
  const res = await fetch(`${BASE}/${method}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify(body),
  });
  if (!res.ok || !res.body) throw new Error(`${method} stream failed (${res.status})`);
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    const lines = buf.split("\n");
    buf = lines.pop() ?? "";
    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed) yield JSON.parse(trimmed) as O;
    }
  }
}

export const apiClient = {
  listRuns:      (req: Partial<ListRunsRequest>) => call<Partial<ListRunsRequest>, RunList>("ListRuns", req),
  getRun:        (req: RunID)                    => call<RunID, RunStatus>("GetRun", req),
  cancelRun:     (req: RunID)                    => call<RunID, Empty>("CancelRun", req),
  listAgents:    (req: Empty)                    => call<Empty, AgentList>("ListAgents", req),
  listWorkflows: (req: Empty)                    => call<Empty, WorkflowList>("ListWorkflows", req),
  getWorkflow:   (req: WorkflowRef)              => call<WorkflowRef, WorkflowDAG>("GetWorkflow", req),
  streamLogs:    (req: RunID)                    => stream<RunID, LogChunk>("StreamLogs", req),
};
