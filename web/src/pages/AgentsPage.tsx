import React from "react";
import { useAgents } from "../hooks/useAgents";

export function AgentsPage() {
  const { agents, loading, error, refresh } = useAgents();

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>Agents</h2>
        <button onClick={refresh} style={btnStyle}>Refresh</button>
      </div>

      {loading && <p style={{ color: "#6b7280" }}>Loading…</p>}
      {error && <p style={{ color: "#ef4444" }}>Error: {error}</p>}

      <div style={{ display: "grid", gap: 12 }}>
        {agents.map((agent) => (
          <div key={agent.agentId} style={{
            border: "1px solid #e5e7eb", borderRadius: 6, padding: "12px 16px",
            background: "#fff", display: "flex", gap: 16, alignItems: "center",
          }}>
            <div>
              <div style={{ fontWeight: 600 }}>{agent.name}</div>
              <div style={{ fontSize: 12, color: "#6b7280", fontFamily: "monospace" }}>{agent.agentId}</div>
            </div>
            <div style={{ marginLeft: "auto", display: "flex", gap: 16, fontSize: 13 }}>
              <span>
                <strong>{agent.active}</strong>/{agent.capacity} jobs
              </span>
              {agent.labels.length > 0 && (
                <span style={{ color: "#6b7280" }}>{agent.labels.join(", ")}</span>
              )}
              <span style={{ color: agent.active < agent.capacity ? "#22c55e" : "#f59e0b", fontWeight: 600 }}>
                {agent.active < agent.capacity ? "available" : "busy"}
              </span>
            </div>
          </div>
        ))}
        {!loading && agents.length === 0 && (
          <p style={{ color: "#9ca3af" }}>No agents connected.</p>
        )}
      </div>
    </div>
  );
}

const btnStyle: React.CSSProperties = {
  padding: "6px 14px", background: "#3b82f6", color: "#fff",
  border: "none", borderRadius: 4, cursor: "pointer", fontWeight: 600, fontSize: 13,
};
