import { AppEntry, AppSource } from "../types/app";
import {
  backendSearch,
  backendGetApp,
  backendGetPopular,
  backendInstall,
  backendRemove,
} from "../ipc/backend";

export async function fetchApps(query: {
  q?: string;
  limit?: number;
  offset?: number;
}): Promise<{ apps: AppEntry[]; total: number }> {
  if (!query.q) return { apps: [], total: 0 };
  const results = await backendSearch(query.q, () => {});
  const offset = query.offset ?? 0;
  const limit = query.limit ?? results.length;
  return {
    apps: results.slice(offset, offset + limit),
    total: results.length,
  };
}

export async function fetchApp(id: string): Promise<AppEntry> {
  return backendGetApp(id);
}

export async function fetchPopularApps(): Promise<AppEntry[]> {
  return backendGetPopular();
}

export async function installApp(source: AppSource): Promise<void> {
  return backendInstall(source.type, source.packageName);
}

export async function removeApp(source: AppSource): Promise<void> {
  return backendRemove(source.type, source.packageName);
}
