import React, { useEffect, useState } from "react";
import type { WorkflowDAG } from "../lib/brazier_connect";
import { DAGView } from "../components/DAGView";

export function WorkflowsPage() {
  const [names, setNames] = useState<string[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [dag, setDag] = useState<WorkflowDAG | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function load() {
      try {
        const { apiClient } = await import("../lib/client");
        const resp = await apiClient.listWorkflows({});
        setNames(resp.names ?? []);
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      }
    }
    void load();
  }, []);

  async function selectWorkflow(name: string) {
    setSelected(name);
    setDag(null);
    setLoading(true);
    try {
      const { apiClient } = await import("../lib/client");
      const resp = await apiClient.getWorkflow({ name, version: "latest" });
      setDag(resp);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <h2 style={{ marginTop: 0 }}>Workflows</h2>
      {error && <p style={{ color: "#ef4444" }}>Error: {error}</p>}

      <div style={{ display: "flex", gap: 16 }}>
        <div style={{ width: 200, flexShrink: 0 }}>
          {names.map((name) => (
            <button key={name} onClick={() => void selectWorkflow(name)} style={{
              display: "block", width: "100%", textAlign: "left",
              padding: "8px 12px", marginBottom: 4,
              background: selected === name ? "#eff6ff" : "transparent",
              border: selected === name ? "1px solid #bfdbfe" : "1px solid transparent",
              borderRadius: 4, cursor: "pointer", fontFamily: "monospace", fontSize: 13,
            }}>
              {name}
            </button>
          ))}
          {names.length === 0 && <p style={{ color: "#9ca3af", fontSize: 13 }}>No workflows.</p>}
        </div>

        <div style={{ flex: 1 }}>
          {loading && <p style={{ color: "#6b7280" }}>Loading…</p>}
          {dag && (
            <>
              <div style={{ marginBottom: 12, fontSize: 13, color: "#6b7280" }}>
                {dag.name} @ {dag.version}
              </div>
              <DAGView nodes={dag.nodes} />
            </>
          )}
          {!selected && <p style={{ color: "#9ca3af", fontSize: 13 }}>Select a workflow to inspect its DAG.</p>}
        </div>
      </div>
    </div>
  );
}
