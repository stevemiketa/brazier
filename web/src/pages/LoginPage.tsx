import React, { useState } from "react";
import { setApiKey } from "../lib/auth";

interface Props {
  onLogin: () => void;
}

export function LoginPage({ onLogin }: Props) {
  const [key, setKey] = useState("");
  const [error, setError] = useState<string | null>(null);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = key.trim();
    if (!trimmed.startsWith("bz_")) {
      setError("API key must start with bz_");
      return;
    }
    setApiKey(trimmed);
    onLogin();
  }

  return (
    <div style={{ display: "flex", alignItems: "center", justifyContent: "center", minHeight: "100vh", background: "#f8fafc" }}>
      <div style={{ background: "#fff", borderRadius: 8, padding: 32, boxShadow: "0 1px 4px rgba(0,0,0,0.1)", width: 360 }}>
        <h1 style={{ margin: "0 0 24px", fontSize: 22, fontWeight: 700 }}>🔥 Brazier CI</h1>
        <form onSubmit={handleSubmit}>
          <label style={{ display: "block", fontSize: 13, fontWeight: 600, marginBottom: 6 }}>API Key</label>
          <input
            type="password"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="bz_..."
            style={{ width: "100%", padding: "8px 10px", border: "1px solid #d1d5db", borderRadius: 4, fontSize: 14, boxSizing: "border-box" }}
          />
          {error && <p style={{ color: "#ef4444", fontSize: 13, margin: "8px 0 0" }}>{error}</p>}
          <button type="submit" style={{
            marginTop: 16, width: "100%", padding: "9px 0",
            background: "#3b82f6", color: "#fff", border: "none",
            borderRadius: 4, fontWeight: 600, fontSize: 14, cursor: "pointer",
          }}>
            Sign in
          </button>
        </form>
      </div>
    </div>
  );
}
