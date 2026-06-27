import { useState, useEffect, useCallback } from "react";
import type { RunStatus } from "../lib/brazier_connect";
import { apiClient } from "../lib/client";

// In production these fetch via the gRPC-web client.
// We export a thin hook so pages don't depend on transport details.

export interface UseRunsResult {
  runs: RunStatus[];
  loading: boolean;
  error: string | null;
  refresh: () => void;
}

export function useRuns(project: string, limit = 20): UseRunsResult {
  const [runs, setRuns] = useState<RunStatus[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await apiClient.listRuns({ project, limit });
      setRuns(resp.runs ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [project, limit]);

  useEffect(() => { void fetch(); }, [fetch]);

  return { runs, loading, error, refresh: fetch };
}
