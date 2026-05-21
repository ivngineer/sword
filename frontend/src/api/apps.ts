import { AppEntry } from "../types/app";
import { MOCK_APPS } from "../mock/apps";

const delay = () => new Promise<void>((r) => setTimeout(r, 300));

export async function fetchApps(query: {
  q?: string;
  limit?: number;
  offset?: number;
}): Promise<{ apps: AppEntry[]; total: number }> {
  await delay();
  let results = MOCK_APPS;
  if (query.q) {
    const q = query.q.toLowerCase();
    results = results.filter((a) => a.name.toLowerCase().includes(q));
  }
  const offset = query.offset ?? 0;
  const limit = query.limit ?? results.length;
  const page = results.slice(offset, offset + limit);
  return { apps: page, total: results.length };
}

export async function fetchApp(id: string): Promise<AppEntry> {
  await delay();
  const app = MOCK_APPS.find((a) => a.id === id);
  if (!app) throw new Error(`App not found: ${id}`);
  return app;
}
