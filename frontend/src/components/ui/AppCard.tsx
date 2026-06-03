import { useState } from "react";
import { Button } from "@heroui/react";
import { CircuitBoard, Cpu, Terminal } from "lucide-react";
import { AppEntry } from "../../types/app";
import { SourceSwitcher } from "./SourceSwitcher";
import { useAppSources } from "../../hooks/useAppSources";
import { useUIStore } from "../../store/ui.store";

// driverLabel maps the backend's DriverKind to a friendly, non-technical chip
// label. Returns null for non-driver entries.
function driverLabel(kind: AppEntry["driverKind"]): string | null {
  switch (kind) {
    case "kernel-module":
      return "Kernel Module";
    case "firmware":
      return "Firmware";
    case "driver":
      return "Driver";
    default:
      return null;
  }
}

// driverIcon picks a category glyph per driver kind. Drivers rarely ship a
// desktop entry / icon, so we stand in a recognizable symbol instead:
// kernel-module → Terminal (loaded via shell, software), firmware → Cpu (chip),
// driver → CircuitBoard (hardware). Returns null for non-driver entries.
export function driverIcon(kind: AppEntry["driverKind"]) {
  switch (kind) {
    case "kernel-module":
      return Terminal;
    case "firmware":
      return Cpu;
    case "driver":
      return CircuitBoard;
    default:
      return null;
  }
}

export function AppCard({ entry }: { entry: AppEntry }) {
  const [imgError, setImgError] = useState(false);
  const { activeSource, allSources, setSource } = useAppSources(entry);
  const installed = entry.status === "installed";
  const viewApp = useUIStore((s) => s.viewApp);
  const badge = driverLabel(entry.driverKind);
  const KindIcon = driverIcon(entry.driverKind);

  // Driver cards use a distinct layout: the name spans the full width up top
  // (no icon pushing it right), and a small category glyph sits beside the
  // 2-line description so everything stays vertically aligned.
  if (KindIcon) {
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
        {/* Title row: full width, badge at end */}
        <div className="flex items-center gap-2 min-w-0">
          <h3
            className="text-[22px] font-semibold leading-tight truncate select-none flex-1 min-w-0"
            style={{ color: "var(--foreground)" }}
          >
            {entry.name}
          </h3>
          {badge && (
            <span
              className="shrink-0 rounded-full px-2 py-0.5 text-xs font-medium select-none"
              style={{ backgroundColor: "var(--surface)", color: "var(--muted)" }}
            >
              {badge}
            </span>
          )}
        </div>

        {/* Icon + description: small glyph sized to the 2-line description */}
        <div className="flex flex-row gap-4 items-start flex-1 min-h-0">
          <div
            className="w-[44px] h-[44px] shrink-0 rounded-lg flex items-center justify-center"
            style={{ backgroundColor: "var(--surface)" }}
          >
            <KindIcon size={24} style={{ color: "var(--muted)" }} />
          </div>
          <p
            className="text-sm line-clamp-2 select-none"
            style={{ color: "var(--muted)" }}
          >
            {entry.description}
          </p>
        </div>

        {/* Bottom row: source dropdown + Get button */}
        <div
          className="flex items-center gap-3"
          onClick={(e) => e.stopPropagation()}
          onKeyDown={(e) => e.stopPropagation()}
        >
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
          <div className="flex items-center gap-2 min-w-0">
            <h3
              className="text-[22px] font-semibold leading-tight truncate select-none"
              style={{ color: "var(--foreground)" }}
            >
              {entry.name}
            </h3>
            {badge && (
              <span
                className="shrink-0 rounded-full px-2 py-0.5 text-xs font-medium select-none"
                style={{ backgroundColor: "var(--surface)", color: "var(--muted)" }}
              >
                {badge}
              </span>
            )}
          </div>
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
