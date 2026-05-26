import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@heroui/react";
import { ArrowLeft, Loader2 } from "lucide-react";
import { useUIStore } from "../store/ui.store";
import { fetchApp, installApp, removeApp } from "../api/apps";
import { useAppSources } from "../hooks/useAppSources";
import { SourceSwitcher } from "../components/ui/SourceSwitcher";
import { formatBytes } from "../lib/format";
import { AppEntry } from "../types/app";
import { tokens } from "../theme/tokens";

export function AppScreen() {
  const snapshot = useUIStore((s) => s.activeAppEntry);
  const id = useUIStore((s) => s.activeAppId);
  const closeApp = useUIStore((s) => s.closeApp);

  // Snapshot renders immediately; a background fetch enriches with screenshots
  // and any field missing from the partial entry that came through search.
  const { data: fresh } = useQuery({
    queryKey: ["app", id],
    queryFn: () => fetchApp(id!),
    enabled: !!id,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") closeApp();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [closeApp]);

  if (!snapshot) return null;
  // Prefer the fresh entry once it arrives, but keep snapshot screenshots if
  // the fresh entry lacks them (e.g. backend index not yet rebuilt).
  const entry: AppEntry = fresh
    ? {
        ...fresh,
        iconUrl: fresh.iconUrl || snapshot.iconUrl,
        screenshots: fresh.screenshots ?? snapshot.screenshots,
      }
    : snapshot;

  return <AppScreenContent entry={entry} onBack={closeApp} />;
}

function AppScreenContent({ entry, onBack }: { entry: AppEntry; onBack: () => void }) {
  const { activeSource, allSources, setSource } = useAppSources(entry);
  const qc = useQueryClient();
  const [busy, setBusy] = useState<null | "install" | "remove">(null);

  const installed = activeSource.installed;
  const mutate = useMutation({
    mutationFn: async (action: "install" | "remove") => {
      if (action === "install") await installApp(activeSource);
      else await removeApp(activeSource);
    },
    onMutate: (action) => setBusy(action),
    onSuccess: (_data, action) => {
      // Backend rebuilds the index asynchronously after the action completes,
      // so an immediate refetch can still return the stale installed state.
      // Patch the cached entry by hand so the button flips instantly, then
      // let the eventual invalidation reconcile.
      const installedNext = action === "install";
      qc.setQueryData<AppEntry>(["app", entry.id], (prev) => {
        const base = prev ?? entry;
        const sources = base.sources.map((s) =>
          s.id === activeSource.id ? { ...s, installed: installedNext } : s,
        );
        const anyInstalled = sources.some((s) => s.installed);
        return {
          ...base,
          sources,
          status: anyInstalled ? "installed" : "available",
        };
      });
    },
    onSettled: () => {
      setBusy(null);
      qc.invalidateQueries({ queryKey: ["app", entry.id] });
      qc.invalidateQueries({ queryKey: ["popular"] });
    },
  });

  const screenshots = entry.screenshots ?? [];
  const sizeLabel = formatBytes(activeSource.sizeBytes);

  return (
    <div
      className="flex flex-col w-full h-full overflow-y-auto"
      style={{ padding: tokens.spacing.outer }}
    >
      {/* Back */}
      <div className="mb-5">
        <button
          onClick={onBack}
          className="flex items-center gap-2 text-sm rounded-lg px-3 py-2"
          style={{ color: "var(--muted)", backgroundColor: "var(--surface-secondary)" }}
        >
          <ArrowLeft size={16} />
          Back
        </button>
      </div>

      {/* Header */}
      <div className="flex flex-row gap-6 items-start">
        <div className="w-[128px] h-[128px] shrink-0 flex items-center justify-center">
          <AppIcon entry={entry} />
        </div>
        <div className="flex flex-col gap-2 flex-1 min-w-0 pt-1">
          <h1
            className="text-4xl font-semibold leading-tight"
            style={{ color: "var(--foreground)" }}
          >
            {entry.name}
          </h1>
          {entry.publisher && (
            <p className="text-sm" style={{ color: "var(--muted)" }}>
              {entry.publisher}
            </p>
          )}
          <div className="flex items-center gap-3 mt-3 max-w-md">
            <SourceSwitcher
              sources={allSources}
              value={activeSource}
              onChange={setSource}
            />
            <ActionButton
              installed={installed}
              busy={busy}
              onClick={() => mutate.mutate(installed ? "remove" : "install")}
            />
          </div>
          {mutate.isError && (
            <p className="text-xs mt-2" style={{ color: "#ef4444" }}>
              {(mutate.error as Error).message}
            </p>
          )}
        </div>
      </div>

      {/* Screenshots */}
      {screenshots.length > 0 && (
        <div className="mt-8 -mx-6 px-6 overflow-x-auto">
          <div className="flex flex-row gap-3">
            {screenshots.map((src, i) => (
              <img
                key={i}
                src={src}
                alt={`${entry.name} screenshot ${i + 1}`}
                loading="lazy"
                draggable={false}
                className="rounded-xl shrink-0 object-cover"
                style={{
                  width: 360,
                  height: 220,
                  backgroundColor: "var(--surface-secondary)",
                }}
              />
            ))}
          </div>
        </div>
      )}

      {/* Description */}
      {entry.description && (
        <p
          className="mt-8 text-base leading-relaxed whitespace-pre-line"
          style={{ color: "var(--foreground)" }}
        >
          {entry.description}
        </p>
      )}

      {/* Metadata */}
      <div className="mt-8 grid grid-cols-2 sm:grid-cols-3 gap-4 max-w-xl">
        <Meta label="Source" value={activeSource.type} />
        <Meta label="Package" value={activeSource.packageName} />
        <Meta label="Version" value={activeSource.version || "—"} />
        {sizeLabel && <Meta label="Size" value={sizeLabel} />}
      </div>
    </div>
  );
}

function AppIcon({ entry }: { entry: AppEntry }) {
  const [err, setErr] = useState(false);
  if (err || !entry.iconUrl) {
    return (
      <span className="text-sm" style={{ color: "var(--muted)" }}>
        Icon
      </span>
    );
  }
  return (
    <img
      src={entry.iconUrl}
      alt={entry.name}
      onError={() => setErr(true)}
      draggable={false}
      className="w-full h-full object-contain"
    />
  );
}

function ActionButton({
  installed,
  busy,
  onClick,
}: {
  installed: boolean;
  busy: null | "install" | "remove";
  onClick: () => void;
}) {
  const isBusy = busy !== null;
  let bg = "#3b82f6"; // blue (install)
  let label = "Install";
  if (installed) {
    bg = "#ef4444"; // red (remove)
    label = "Remove";
  }
  if (busy === "install") label = "Installing…";
  if (busy === "remove") label = "Removing…";
  if (isBusy) bg = "#6b7280";

  return (
    <Button
      variant="secondary"
      size="sm"
      isDisabled={isBusy}
      onPress={onClick}
      className="rounded-full px-5 shrink-0"
      style={{
        backgroundColor: bg,
        color: "#ffffff",
        opacity: isBusy ? 0.85 : 1,
      }}
    >
      <span className="inline-flex items-center gap-2">
        {isBusy && <Loader2 size={14} className="animate-spin" />}
        {label}
      </span>
    </Button>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs uppercase tracking-wide" style={{ color: "var(--muted)" }}>
        {label}
      </span>
      <span className="text-sm" style={{ color: "var(--foreground)" }}>
        {value}
      </span>
    </div>
  );
}
