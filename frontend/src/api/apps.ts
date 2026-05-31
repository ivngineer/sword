import { AppEntry, AppSource } from "../types/app";
import {
  backendSearch,
  backendGetApp,
  backendGetPopular,
  backendListInstalled,
  backendInstall,
  backendRemove,
} from "../ipc/backend";
import { useProgressStore } from "../store/progress.store";

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

export async function fetchInstalledApps(): Promise<AppEntry[]> {
  return backendListInstalled();
}

export async function installApp(source: AppSource, appName: string): Promise<void> {
  return runAction("install", source, appName);
}

export async function removeApp(source: AppSource, appName: string): Promise<void> {
  return runAction("remove", source, appName);
}

// runAction wires a backend install/remove into the global progress store so
// the sidebar bar reflects whatever's currently happening regardless of which
// screen kicked it off.
async function runAction(
  kind: "install" | "remove",
  source: AppSource,
  appName: string,
): Promise<void> {
  const { start, update, finish } = useProgressStore.getState();
  const fn = kind === "install" ? backendInstall : backendRemove;
  const { id, done } = await fn(source.type, source.packageName, ({ fraction, status }) => {
    update(id, fraction, status);
  });
  start({ id, kind, appName, sourceType: source.type });
  try {
    await done;
  } finally {
    finish(id);
  }
}
