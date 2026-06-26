import React, { useState } from "react";
import { useRuns } from "../hooks/useRuns";
import { StatusBadge } from "../components/StatusBadge";

interface Props {
  onSelectRun: (runId: string) => void;
}

export function DashboardPage({ onSelectRun }: Props) {
  const [project, setProject] = useState("");
  const { runs, loading, error, refresh } = useRuns(project);

  return (
    <div>
      <div style={{ display: "flex", gap: 8, marginBottom: 16, alignItems: "center" }}>
        <input
          value={project}
          onChange={(e) => setProject(e.target.value)}
          placeholder="Filter by project…"
          style={{ padding: "6px 10px", border: "1px solid #d1d5db", borderRadius: 4, fontSize: 14, flex: 1 }}
        />
        <button onClick={refresh} style={btnStyle}>Refresh</button>
      </div>

      {loading && <p style={{ color: "#6b7280" }}>Loading…</p>}
      {error && <p style={{ color: "#ef4444" }}>Error: {error}</p>}

      <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 14 }}>
        <thead>
          <tr style={{ borderBottom: "2px solid #e5e7eb", textAlign: "left" }}>
            <th style={th}>Run ID</th>
            <th style={th}>Project</th>
            <th style={th}>State</th>
            <th style={th}></th>
          </tr>
        </thead>
        <tbody>
          {runs.map((run) => (
            <tr key={run.runId} style={{ borderBottom: "1px solid #f1f5f9" }}>
              <td style={td}><code>{run.runId.slice(0, 12)}…</code></td>
              <td style={td}>{run.runId}</td>
              <td style={td}><StatusBadge state={run.state} /></td>
              <td style={td}>
                <button onClick={() => onSelectRun(run.runId)} style={linkBtn}>View</button>
              </td>
            </tr>
          ))}
          {!loading && runs.length === 0 && (
            <tr><td colSpan={4} style={{ ...td, color: "#9ca3af" }}>No runs found.</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

const th: React.CSSProperties = { padding: "8px 12px", fontWeight: 600, color: "#374151" };
const td: React.CSSProperties = { padding: "8px 12px", color: "#1f2937" };
const btnStyle: React.CSSProperties = {
  padding: "6px 14px", background: "#3b82f6", color: "#fff",
  border: "none", borderRadius: 4, cursor: "pointer", fontWeight: 600, fontSize: 13,
};
const linkBtn: React.CSSProperties = {
  ...btnStyle, background: "transparent", color: "#3b82f6", padding: "4px 8px",
};
