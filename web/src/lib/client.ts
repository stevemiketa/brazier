import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { BrazierAPI } from "./brazier_connect";

const transport = createConnectTransport({
  baseUrl: window.location.origin,
});

export const apiClient = createClient(BrazierAPI, transport);

export function getAuthHeaders(): Record<string, string> {
  const key = localStorage.getItem("brazier_api_key") ?? "";
  return key ? { Authorization: `Bearer ${key}` } : {};
}
