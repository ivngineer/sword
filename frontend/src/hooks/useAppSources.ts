import { AppEntry, AppSource } from "../types/app";
import { useUIStore } from "../store/ui.store";
import { rankSources } from "../lib/source-rank";

export function useAppSources(entry: AppEntry): {
  activeSource: AppSource;
  allSources: AppSource[];
  setSource: (sourceId: string) => void;
} {
  const overrides = useUIStore((s) => s.sourceOverrides);
  const setSourceOverride = useUIStore((s) => s.setSourceOverride);

  // Duplicate source ids freeze react-aria collections (infinite render
  // loop in the Select popover), so guard against a misbehaving backend.
  const seen = new Set<string>();
  const sources = entry.sources.filter((s) =>
    seen.has(s.id) ? false : (seen.add(s.id), true),
  );

  const overrideId = overrides[entry.id];
  const activeSource =
    sources.find((s) => s.id === overrideId) ??
    sources.find((s) => s.isRecommended) ??
    rankSources(sources);

  return {
    activeSource,
    allSources: sources,
    setSource: (sourceId) => setSourceOverride(entry.id, sourceId),
  };
}
