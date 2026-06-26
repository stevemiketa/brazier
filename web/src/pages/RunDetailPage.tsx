import React, { useEffect, useState } from "react";
import { StatusBadge } from "../components/StatusBadge";
import { DAGView } from "../components/DAGView";
import { LogStream } from "../components/LogStream";
import type { RunStatus } from "../lib/brazier_connect";

interface Props {
  runId: string;
  onBack: () => void;
}

export function RunDetailPage({ runId, onBack }: Props) {
  const [run, setRun] = useState<RunStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const { apiClient } = await import("../lib/client");
        const resp = await apiClient.getRun({ id: runId });
        if (!cancelled) setRun(resp);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      }
    }
    void load();
    return () => { cancelled = true; };
  }, [runId]);

  return (
    <div>
      <button onClick={onBack} style={{ marginBottom: 16, background: "none", border: "none", color: "#3b82f6", cursor: "pointer", fontSize: 14 }}>
        ← Back to dashboard
      </button>

      {error && <p style={{ color: "#ef4444" }}>Error: {error}</p>}

      {run && (
        <>
          <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 20 }}>
            <h2 style={{ margin: 0, fontSize: 18, fontFamily: "monospace" }}>{run.runId}</h2>
            <StatusBadge state={run.state} />
          </div>

          <h3 style={{ fontSize: 14, fontWeight: 600, marginBottom: 8, color: "#374151" }}>DAG</h3>
          <DAGView
            nodes={run.nodes.map((id) => ({ id, dependsOn: [], conditions: [] }))}
          />

          <h3 style={{ fontSize: 14, fontWeight: 600, margin: "20px 0 8px", color: "#374151" }}>Logs</h3>
          <LogStream runId={runId} />
        </>
      )}
    </div>
  );
}
