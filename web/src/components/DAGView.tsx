import React from "react";
import type { WorkflowNode } from "../lib/brazier_connect";
import { StatusBadge } from "./StatusBadge";

interface Props {
  nodes: WorkflowNode[];
  nodeStates?: Record<string, string>; // nodeID → state
}

/**
 * Simple vertical DAG visualisation. Each node is rendered as a card with
 * its dependencies listed. A production implementation would use a proper
 * graph layout library (e.g. dagre + svg), but this gives a readable
 * representation without external dependencies.
 */
export function DAGView({ nodes, nodeStates = {} }: Props) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      {nodes.map((node) => (
        <div key={node.id} style={{
          border: "1px solid #e5e7eb",
          borderRadius: 6,
          padding: "10px 14px",
          background: "#fff",
          display: "flex",
          alignItems: "center",
          gap: 10,
        }}>
          <span style={{ fontWeight: 600, fontFamily: "monospace", minWidth: 120 }}>{node.id}</span>
          {nodeStates[node.id] && <StatusBadge state={nodeStates[node.id] ?? "pending"} />}
          {node.dependsOn.length > 0 && (
            <span style={{ color: "#6b7280", fontSize: 12 }}>
              depends on: {node.dependsOn.join(", ")}
            </span>
          )}
          {node.conditions.length > 0 && (
            <span style={{ color: "#a78bfa", fontSize: 12, marginLeft: "auto" }}>
              when: {node.conditions.join(", ")}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}
