import React from "react";

const COLORS: Record<string, string> = {
  pending:   "#6b7280",
  running:   "#3b82f6",
  success:   "#22c55e",
  failed:    "#ef4444",
  cancelled: "#f59e0b",
  skipped:   "#a78bfa",
};

interface Props {
  state: string;
}

export function StatusBadge({ state }: Props) {
  const color = COLORS[state] ?? "#6b7280";
  return (
    <span style={{
      display: "inline-block",
      padding: "2px 8px",
      borderRadius: 4,
      fontSize: 12,
      fontWeight: 600,
      color: "#fff",
      backgroundColor: color,
      textTransform: "uppercase",
      letterSpacing: "0.05em",
    }}>
      {state}
    </span>
  );
}
