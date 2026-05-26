import { Command, Child } from "@tauri-apps/plugin-shell";
import { convertFileSrc } from "@tauri-apps/api/core";
import { AppEntry } from "../types/app";

// Outbound messages emitted by the Go sidecar (see backend/main.go).
type Outbound =
  | { type: "search_results"; id: string; phase: SearchPhase; results: AppEntry[] }
  | { type: "app_detail"; id: string; app: AppEntry }
  | { type: "popular_results"; id: string; results: AppEntry[] }
  | { type: "install_result"; id: string; ok: boolean }
  | { type: "remove_result"; id: string; ok: boolean }
  | { type: "error"; id: string; message: string };

export type SearchPhase = "local" | "complete";

type Handler = (msg: Outbound) => void;

const pending = new Map<string, Handler>();
let child: Child | null = null;
let starting: Promise<void> | null = null;
let stdoutBuf = "";
let counter = 0;

function nextId(prefix: string): string {
  counter += 1;
  return `${prefix}-${counter}`;
}

// Local file icons (file://) cannot be loaded directly by the webview; they
// must go through Tauri's asset protocol.
function normalizeIcon(url: string): string {
  if (url && url.startsWith("file://")) {
    return convertFileSrc(url.slice("file://".length));
  }
  return url;
}

function dispatch(line: string) {
  const trimmed = line.trim();
  if (!trimmed) return;
  let msg: Outbound;
  try {
    msg = JSON.parse(trimmed);
  } catch {
    console.error("[backend] non-JSON line:", trimmed);
    return;
  }
  if (msg.type === "search_results" || msg.type === "popular_results") {
    for (const r of msg.results) r.iconUrl = normalizeIcon(r.iconUrl);
  } else if (msg.type === "app_detail" && msg.app) {
    msg.app.iconUrl = normalizeIcon(msg.app.iconUrl);
  }
  const handler = pending.get(msg.id);
  if (handler) handler(msg);
}

// ensureStarted spawns the sidecar once and wires up its stdout line reader.
async function ensureStarted(): Promise<void> {
  if (child) return;
  if (starting) return starting;
  starting = (async () => {
    const cmd = Command.sidecar("binaries/sword-backend");
    cmd.stdout.on("data", (chunk: string) => {
      stdoutBuf += chunk;
      let nl: number;
      while ((nl = stdoutBuf.indexOf("\n")) >= 0) {
        const line = stdoutBuf.slice(0, nl);
        stdoutBuf = stdoutBuf.slice(nl + 1);
        dispatch(line);
      }
    });
    cmd.stderr.on("data", (chunk: string) => console.error("[backend]", chunk));
    cmd.on("close", () => {
      child = null;
      starting = null;
      for (const handler of pending.values()) {
        handler({ type: "error", id: "", message: "backend process exited" });
      }
      pending.clear();
    });
    cmd.on("error", (err) => console.error("[backend] spawn error:", err));
    child = await cmd.spawn();
  })();
  return starting;
}

function send(msg: object) {
  if (!child) throw new Error("backend not started");
  child.write(JSON.stringify(msg) + "\n");
}

// backendSearch runs a two-phase search. onPhase fires once for the fast
// local results and once for the complete (AUR-merged) results. The promise
// resolves with the final results, or with [] if signal aborts first.
export async function backendSearch(
  query: string,
  onPhase: (phase: SearchPhase, results: AppEntry[]) => void,
  signal?: AbortSignal,
): Promise<AppEntry[]> {
  await ensureStarted();
  const id = nextId("search");
  return new Promise<AppEntry[]>((resolve, reject) => {
    let settled = false;
    const finish = (fn: () => void) => {
      if (settled) return;
      settled = true;
      pending.delete(id);
      fn();
    };
    if (signal) {
      if (signal.aborted) {
        finish(() => resolve([]));
        return;
      }
      signal.addEventListener("abort", () => finish(() => resolve([])), {
        once: true,
      });
    }
    pending.set(id, (msg) => {
      if (msg.type === "error") {
        finish(() => reject(new Error(msg.message)));
        return;
      }
      if (msg.type === "search_results") {
        onPhase(msg.phase, msg.results);
        if (msg.phase === "complete") {
          finish(() => resolve(msg.results));
        }
      }
    });
    send({ type: "search", id, query });
  });
}

// backendGetPopular returns the popular apps list from the index.
export async function backendGetPopular(): Promise<AppEntry[]> {
  await ensureStarted();
  const id = nextId("popular");
  return new Promise<AppEntry[]>((resolve, reject) => {
    pending.set(id, (msg) => {
      pending.delete(id);
      if (msg.type === "popular_results") resolve(msg.results);
      else if (msg.type === "error") reject(new Error(msg.message));
      else reject(new Error("unexpected backend response"));
    });
    send({ type: "get_popular", id });
  });
}

// backendInstall runs install for one package via the named source. Resolves
// when the backend finishes the action; rejects on backend error.
export async function backendInstall(sourceType: string, packageName: string): Promise<void> {
  await ensureStarted();
  const id = nextId("install");
  return new Promise<void>((resolve, reject) => {
    pending.set(id, (msg) => {
      pending.delete(id);
      if (msg.type === "install_result") resolve();
      else if (msg.type === "error") reject(new Error(msg.message));
      else reject(new Error("unexpected backend response"));
    });
    send({ type: "install", id, source_type: sourceType, package_name: packageName });
  });
}

// backendRemove uninstalls one package via the named source.
export async function backendRemove(sourceType: string, packageName: string): Promise<void> {
  await ensureStarted();
  const id = nextId("remove");
  return new Promise<void>((resolve, reject) => {
    pending.set(id, (msg) => {
      pending.delete(id);
      if (msg.type === "remove_result") resolve();
      else if (msg.type === "error") reject(new Error(msg.message));
      else reject(new Error("unexpected backend response"));
    });
    send({ type: "remove", id, source_type: sourceType, package_name: packageName });
  });
}

// backendGetApp fetches full detail for a single app by canonical id.
export async function backendGetApp(appId: string): Promise<AppEntry> {
  await ensureStarted();
  const id = nextId("get");
  return new Promise<AppEntry>((resolve, reject) => {
    pending.set(id, (msg) => {
      pending.delete(id);
      if (msg.type === "app_detail") resolve(msg.app);
      else if (msg.type === "error") reject(new Error(msg.message));
      else reject(new Error("unexpected backend response"));
    });
    send({ type: "get_app", id, app_id: appId });
  });
}
