import { useEffect, useState } from "react";
import { Button } from "@heroui/react";
import { Loader2, Minus, RefreshCw, Undo2 } from "lucide-react";
import { UpdateEntry } from "../types/app";
import { useUpdatesStore, skipKey } from "../store/updates.store";
import { SourceBadge } from "../components/ui/SourceBadge";

export function UpdatesScreen() {
  const { entries, loaded, loading, updating, error, skipped, refresh, runFullUpdate, cancelUpdate } =
    useUpdatesStore();

  // Launch prefetch usually covers this; the guard avoids a double fetch.
  useEffect(() => {
    if (!loaded && !loading) refresh();
  }, [loaded, loading, refresh]);

  const [scrolled, setScrolled] = useState(false);
  const [updateBtnHovered, setUpdateBtnHovered] = useState(false);

  const skipCount = entries.filter((e) => skipped[skipKey(e)]).length;
  const allSkipped = entries.length > 0 && skipCount === entries.length;

  return (
    <div className="flex flex-1 flex-col items-center justify-center overflow-hidden">
      <div className="flex w-full max-w-[560px] flex-col gap-3 max-h-full py-2">
        {loading && entries.length === 0 ? (
          <span className="text-sm text-center" style={{ color: "var(--muted)" }}>
            Checking for updates…
          </span>
        ) : entries.length === 0 ? (
          <div className="flex flex-col items-center gap-3" style={{ color: "var(--muted)" }}>
            <RefreshCw size={28} />
            <span className="text-sm">Everything is up to date</span>
            <Button size="sm" variant="ghost" onPress={refresh} isDisabled={loading}>
              Check again
            </Button>
          </div>
        ) : (
          <>
            <span className="text-sm px-1" style={{ color: "var(--muted)" }}>
              {entries.length} update{entries.length === 1 ? "" : "s"} available
              {skipCount > 0 ? ` · ${skipCount} skipped` : ""}
            </span>
            <div className="relative min-h-0">
              <div
                className="flex flex-col gap-2 overflow-y-auto min-h-0 max-h-full"
                onScroll={(e) => setScrolled((e.currentTarget as HTMLElement).scrollTop > 0)}
              >
                {entries.map((entry) => (
                  <UpdateTile
                    key={skipKey(entry)}
                    entry={entry}
                    skipped={!!skipped[skipKey(entry)]}
                  />
                ))}
              </div>
              {scrolled && (
                <div
                  className="pointer-events-none absolute top-0 left-0 right-0 h-10"
                  style={{
                    background: "linear-gradient(to top, transparent, var(--background))",
                  }}
                />
              )}
              <div
                className="pointer-events-none absolute bottom-0 left-0 right-0 h-10"
                style={{
                  background: "linear-gradient(to bottom, transparent, var(--background))",
                }}
              />
            </div>
          </>
        )}

        {error && (
          <span className="text-xs px-1" style={{ color: "#ef4444" }} title={error}>
            {error}
          </span>
        )}
        {entries.length > 0 && (
          <Button
            variant="secondary"
            className="w-full rounded-xl py-4 transition-colors"
            onPress={updating ? cancelUpdate : runFullUpdate}
            isDisabled={!updating && allSkipped}
            onMouseEnter={() => setUpdateBtnHovered(true)}
            onMouseLeave={() => setUpdateBtnHovered(false)}
            style={{
              backgroundColor:
                updating && updateBtnHovered
                  ? "rgba(239, 68, 68, 0.85)"
                  : updating
                    ? "#6b7280"
                    : allSkipped
                      ? "#6b7280"
                      : "#3b82f6",
              color: "#ffffff",
              opacity: !updating && allSkipped ? 0.6 : 1,
              minHeight: "3rem",
            }}
          >
            <span className="flex items-center gap-2">
              {updating && !(updating && updateBtnHovered) && (
                <Loader2 size={16} className="animate-spin" />
              )}
              {updating
                ? updateBtnHovered
                  ? "Cancel"
                  : "Updating system"
                : "Full system update"}
            </span>
          </Button>
        )}
      </div>
    </div>
  );
}

// UpdateTile is one pending update. Hover tints it red and reveals a minus
// icon: clicking toggles "skip this update" (session-only). Skipped tiles
// stay red-tinted and dimmed until clicked again.
function UpdateTile({ entry, skipped }: { entry: UpdateEntry; skipped: boolean }) {
  const toggleSkip = useUpdatesStore((s) => s.toggleSkip);
  const [hovered, setHovered] = useState(false);
  const red = hovered || skipped;
  return (
    <button
      type="button"
      onClick={() => toggleSkip(entry)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      title={skipped ? "Include in update" : "Skip this update"}
      className="flex w-full items-center gap-3 rounded-xl px-4 py-3 text-left transition-colors cursor-pointer"
      style={{
        backgroundColor: red ? "rgba(239, 68, 68, 0.12)" : "var(--surface-secondary)",
        opacity: skipped ? 0.55 : 1,
      }}
    >
      {entry.iconUrl && (
        <img
          src={entry.iconUrl}
          alt=""
          className="h-8 w-8 shrink-0 rounded-md object-contain"
          onError={(e) => {
            (e.target as HTMLImageElement).style.display = "none";
          }}
        />
      )}
      <span
        className={`truncate text-sm ${skipped ? "line-through" : ""}`}
        style={{ color: "var(--foreground)" }}
      >
        {entry.name}
      </span>
      <SourceBadge type={entry.sourceType} />
      <span
        className="ml-auto shrink-0 tabular-nums text-xs"
        style={{ color: "var(--muted)" }}
      >
        {/* flatpak runtimes often report no version strings */}
        {entry.currentVersion && entry.newVersion && entry.currentVersion !== entry.newVersion
          ? `${entry.currentVersion} → ${entry.newVersion}`
          : entry.newVersion || entry.currentVersion}
      </span>
      <span className="w-4 shrink-0" style={{ color: "#ef4444" }}>
        {skipped ? <Undo2 size={16} /> : hovered ? <Minus size={16} /> : null}
      </span>
    </button>
  );
}
