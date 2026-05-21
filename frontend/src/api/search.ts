import { AppEntry } from "../types/app";
import { fetchApps } from "./apps";

export type SearchResult = AppEntry;

export type SearchResponse = {
  results: SearchResult[];
  total: number;
  query: string;
};

// Replace with real backend call (e.g. Tauri IPC or REST)
export async function searchApps(
  query: string,
  opts: { limit?: number; offset?: number } = {}
): Promise<SearchResponse> {
  const { apps, total } = await fetchApps({ q: query, ...opts });
  return { results: apps, total, query };
}
