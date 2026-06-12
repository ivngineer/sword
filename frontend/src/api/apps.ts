import { AppEntry, AppSource, FirmwarePackage, UpdateEntry, UpdateSkip } from "../types/app";
import {
  backendSearch,
  backendGetApp,
  backendGetPopular,
  backendListInstalled,
  backendListDrivers,
  backendListUpdates,
  backendScanFirmware,
  backendInstall,
  backendInstallBatch,
  backendRemove,
  backendSystemUpdate,
  SearchPhase,
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

// fetchDriversPhased streams driver results: onPhase fires for the instant
// pacman set and again when AUR results are merged in.
export async function fetchDriversPhased(
  onPhase: (phase: SearchPhase, results: AppEntry[]) => void,
): Promise<AppEntry[]> {
  return backendListDrivers(onPhase);
}

// scanFirmware returns firmware packages recommended for this machine's
// hardware that are not installed. Empty = hide the firmware panel.
export async function scanFirmware(): Promise<FirmwarePackage[]> {
  return backendScanFirmware();
}

// installFirmwarePackages batch-installs the pending firmware list in a
// single pacman transaction, wired into the global progress store like any
// other install.
export async function installFirmwarePackages(names: string[]): Promise<void> {
  if (names.length === 0) return;
  const { start, update, finish } = useProgressStore.getState();
  const { id, done } = await backendInstallBatch("pacman", names, ({ fraction, status }) => {
    update(id, fraction, status);
  });
  start({ id, kind: "install", appName: "Firmware updates", sourceType: "pacman" });
  try {
    await done;
  } finally {
    finish(id);
  }
}

// fetchUpdates returns every pending update across all sources.
export async function fetchUpdates(): Promise<UpdateEntry[]> {
  return backendListUpdates();
}

// runSystemUpdate performs the full system update (minus skipped packages),
// wired into the global progress store like installs are. Throws with the
// backend's summary when any package manager fails.
export async function runSystemUpdate(skip: UpdateSkip): Promise<void> {
  const { start, update, finish } = useProgressStore.getState();
  const { id, done } = await backendSystemUpdate(skip, ({ fraction, status }) => {
    update(id, fraction, status);
  });
  start({ id, kind: "update", appName: "System update", sourceType: "pacman" });
  try {
    await done;
  } finally {
    finish(id);
  }
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
