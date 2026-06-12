import { create } from "zustand";
import { UpdateEntry, UpdateSkip } from "../types/app";
import { fetchUpdates, runSystemUpdate } from "../api/apps";

// skipKey identifies an update entry across refreshes. Skips are
// session-only by design: a relaunch re-checks updates with a clean slate.
export const skipKey = (e: UpdateEntry) => `${e.sourceType}:${e.packageName}`;

type UpdatesState = {
  entries: UpdateEntry[];
  loaded: boolean;
  loading: boolean;
  updating: boolean;
  error: string | null;
  skipped: Record<string, true>;
  toggleSkip: (e: UpdateEntry) => void;
  refresh: () => Promise<void>;
  runFullUpdate: () => Promise<void>;
};

export const useUpdatesStore = create<UpdatesState>((set, get) => ({
  entries: [],
  loaded: false,
  loading: false,
  updating: false,
  error: null,
  skipped: {},

  toggleSkip: (e) =>
    set((s) => {
      const key = skipKey(e);
      const skipped = { ...s.skipped };
      if (skipped[key]) delete skipped[key];
      else skipped[key] = true;
      return { skipped };
    }),

  refresh: async () => {
    if (get().loading) return;
    set({ loading: true, error: null });
    try {
      const entries = await fetchUpdates();
      // Drop skip marks for packages that no longer have a pending update.
      const valid = new Set(entries.map(skipKey));
      const skipped = Object.fromEntries(
        Object.keys(get().skipped)
          .filter((k) => valid.has(k))
          .map((k) => [k, true as const]),
      );
      set({ entries, skipped, loaded: true });
    } catch (err) {
      set({ error: String(err instanceof Error ? err.message : err), loaded: true });
    } finally {
      set({ loading: false });
    }
  },

  runFullUpdate: async () => {
    const { entries, skipped, updating } = get();
    if (updating) return;
    const skip: UpdateSkip = { pacman: [], aur: [], flatpak: [] };
    for (const e of entries) {
      if (skipped[skipKey(e)]) skip[e.sourceType].push(e.packageName);
    }
    set({ updating: true, error: null });
    try {
      await runSystemUpdate(skip);
    } catch (err) {
      set({ error: String(err instanceof Error ? err.message : err) });
    } finally {
      set({ updating: false });
      await get().refresh();
    }
  },
}));
