import React, { useEffect, useRef, useState } from "react";
import type { LogChunk } from "../lib/brazier_connect";

interface Props {
  runId: string;
}

export function LogStream({ runId }: Props) {
  const [lines, setLines] = useState<LogChunk[]>([]);
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;

    async function stream() {
      try {
        const { apiClient } = await import("../lib/client");
        for await (const chunk of apiClient.streamLogs({ id: runId })) {
          if (cancelled) break;
          setLines((prev) => [...prev, chunk]);
        }
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      }
    }

    void stream();
    return () => { cancelled = true; };
  }, [runId]);

  // Auto-scroll to bottom when new lines arrive.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines]);

  return (
    <div style={{
      background: "#0f172a",
      color: "#e2e8f0",
      fontFamily: "monospace",
      fontSize: 13,
      padding: 12,
      borderRadius: 6,
      overflowY: "auto",
      maxHeight: 400,
    }}>
      {lines.map((chunk, i) => (
        <div key={i} style={{ color: chunk.stderr ? "#f87171" : "#e2e8f0" }}>
          <span style={{ color: "#64748b", marginRight: 8 }}>[{chunk.jobId}]</span>
          {chunk.line}
        </div>
      ))}
      {error && <div style={{ color: "#f87171" }}>Error: {error}</div>}
      <div ref={bottomRef} />
    </div>
  );
}
