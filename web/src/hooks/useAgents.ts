import { useState, useEffect, useCallback } from "react";
import type { AgentInfo } from "../lib/brazier_connect";
import { apiClient } from "../lib/client";

export interface UseAgentsResult {
  agents: AgentInfo[];
  loading: boolean;
  error: string | null;
  refresh: () => void;
}

export function useAgents(): UseAgentsResult {
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await apiClient.listAgents({});
      setAgents(resp.agents ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void fetch(); }, [fetch]);

  return { agents, loading, error, refresh: fetch };
}
