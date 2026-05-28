import { useState } from "react";
import { Button } from "@heroui/react";
import { AppEntry } from "../../types/app";
import { SourceSwitcher } from "./SourceSwitcher";
import { useAppSources } from "../../hooks/useAppSources";
import { useUIStore } from "../../store/ui.store";

export function AppCard({ entry }: { entry: AppEntry }) {
  const [imgError, setImgError] = useState(false);
  const { activeSource, allSources, setSource } = useAppSources(entry);
  const installed = entry.status === "installed";
  const viewApp = useUIStore((s) => s.viewApp);

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => viewApp(entry)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          viewApp(entry);
        }
      }}
      className="rounded-xl p-5 h-[189px] flex flex-col gap-3 cursor-pointer transition-colors"
      style={{ backgroundColor: "var(--surface-secondary)" }}
    >
      {/* Top row: icon + text */}
      <div className="flex flex-row gap-5 items-start flex-1 min-h-0">
        <div className="w-[100px] h-[100px] shrink-0 flex items-center justify-center">
          {imgError ? (
            <span className="text-sm select-none" style={{ color: "var(--muted)" }}>
              Icon
            </span>
          ) : (
            <img
              src={entry.iconUrl}
              className="w-full h-full object-contain"
              alt={entry.name}
              onError={() => setImgError(true)}
              draggable={false}
            />
          )}
        </div>

        <div className="flex flex-col min-w-0">
          <h3
            className="text-[22px] font-semibold leading-tight truncate select-none"
            style={{ color: "var(--foreground)" }}
          >
            {entry.name}
          </h3>
          <p
            className="text-sm mt-1 line-clamp-2 select-none"
            style={{ color: "var(--muted)" }}
          >
            {entry.description}
          </p>
        </div>
      </div>

      {/* Bottom row: source dropdown starts under icon, Get button at end.
          Stop click propagation so interacting with controls doesn't trigger
          card navigation. */}
      <div
        className="flex items-center gap-3"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => e.stopPropagation()}
      >
        <div className="w-[100px] shrink-0" />
        <SourceSwitcher
          sources={allSources}
          value={activeSource}
          onChange={setSource}
        />
        <Button
          variant="secondary"
          size="sm"
          isDisabled={installed}
          onPress={() => viewApp(entry)}
          className="rounded-full px-5 shrink-0 ml-auto"
          style={{
            backgroundColor: installed ? "#6b7280" : "#3b82f6",
            color: "#ffffff",
            opacity: installed ? 0.6 : 1,
          }}
        >
          {installed ? "Installed" : "Get"}
        </Button>
      </div>
    </div>
  );
}
