import React, { useState } from "react";
import { isAuthenticated, clearApiKey } from "./lib/auth";
import { LoginPage } from "./pages/LoginPage";
import { DashboardPage } from "./pages/DashboardPage";
import { RunDetailPage } from "./pages/RunDetailPage";
import { AgentsPage } from "./pages/AgentsPage";
import { WorkflowsPage } from "./pages/WorkflowsPage";

type Page = "dashboard" | "run" | "agents" | "workflows";

export function App() {
  const [authed, setAuthed] = useState(isAuthenticated());
  const [page, setPage] = useState<Page>("dashboard");
  const [selectedRun, setSelectedRun] = useState<string | null>(null);

  if (!authed) {
    return <LoginPage onLogin={() => setAuthed(true)} />;
  }

  function nav(p: Page) {
    setPage(p);
    setSelectedRun(null);
  }

  return (
    <div style={{ fontFamily: "system-ui, sans-serif", minHeight: "100vh", background: "#f8fafc" }}>
      {/* Sidebar */}
      <div style={{ position: "fixed", top: 0, left: 0, bottom: 0, width: 200, background: "#1e293b", color: "#e2e8f0", display: "flex", flexDirection: "column" }}>
        <div style={{ padding: "20px 16px 12px", fontSize: 18, fontWeight: 700, borderBottom: "1px solid #334155" }}>
          🔥 Brazier CI
        </div>
        {(["dashboard", "agents", "workflows"] as const).map((p) => (
          <button key={p} onClick={() => nav(p)} style={{
            display: "block", width: "100%", textAlign: "left",
            padding: "10px 16px", background: page === p ? "#334155" : "transparent",
            color: "#e2e8f0", border: "none", cursor: "pointer", fontSize: 14,
            textTransform: "capitalize",
          }}>
            {p === "dashboard" ? "🗂 Runs" : p === "agents" ? "⚙ Agents" : "🔀 Workflows"}
          </button>
        ))}
        <button onClick={() => { clearApiKey(); setAuthed(false); }} style={{
          marginTop: "auto", padding: "10px 16px", background: "transparent",
          color: "#94a3b8", border: "none", cursor: "pointer", fontSize: 13, textAlign: "left",
        }}>
          Sign out
        </button>
      </div>

      {/* Main content */}
      <div style={{ marginLeft: 200, padding: 24 }}>
        {page === "dashboard" && !selectedRun && (
          <DashboardPage onSelectRun={(id) => { setSelectedRun(id); setPage("run"); }} />
        )}
        {page === "run" && selectedRun && (
          <RunDetailPage runId={selectedRun} onBack={() => nav("dashboard")} />
        )}
        {page === "agents" && <AgentsPage />}
        {page === "workflows" && <WorkflowsPage />}
      </div>
    </div>
  );
}
