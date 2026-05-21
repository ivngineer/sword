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

  const overrideId = overrides[entry.id];
  const activeSource =
    entry.sources.find((s) => s.id === overrideId) ??
    entry.sources.find((s) => s.isRecommended) ??
    rankSources(entry.sources);

  return {
    activeSource,
    allSources: entry.sources,
    setSource: (sourceId) => setSourceOverride(entry.id, sourceId),
  };
}
