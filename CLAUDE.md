# Sword

A Linux GUI package manager. Search across pacman, the AUR, and Flatpak from one UI.

## Architecture

Three layers:

1. **Frontend** — React 19 + TypeScript + Vite, inside a Tauri 2 shell. Lives in `frontend/`.
2. **Tauri shell** — Rust, `frontend/src-tauri/`. Thin: registers `tauri-plugin-shell` so the
   frontend can spawn the backend as a **sidecar**. No custom Rust commands.
3. **Go backend** — `backend/`, module `sword/backend`. Runs as a Tauri sidecar. Does all the
   real work: queries package sources, builds the search index, resolves icons.

Data flow for a search:

```
SearchScreen (Enter)
  -> api/search.ts  searchApps()
  -> ipc/backend.ts backendSearch()   spawns sidecar once, writes JSON to stdin
  -> [Go] main.go IPC loop -> search orchestrator -> registry / sources
  -> [Go] writes JSON to stdout
  -> ipc/backend.ts correlates by request id, fires phase callbacks
  -> SearchScreen renders AppCard[]
```

The frontend and backend talk **line-delimited JSON over the sidecar's stdin/stdout**.
Backend stdout is reserved for IPC only; all logs go to stderr.

### IPC protocol

Inbound (frontend -> Go):
```json
{ "type": "search",  "id": "search-1", "query": "firefox" }
{ "type": "get_app", "id": "get-1",    "app_id": "org.mozilla.firefox" }
```

Outbound (Go -> frontend):
```json
{ "type": "search_results", "id": "search-1", "phase": "local",    "results": [...] }
{ "type": "search_results", "id": "search-1", "phase": "complete", "results": [...] }
{ "type": "app_detail",     "id": "get-1",    "app": {...} }
{ "type": "error",          "id": "search-1", "message": "..." }
```

**Two-phase search**: `phase: "local"` is the fast in-memory result (pacman + flatpak from the
index); `phase: "complete"` arrives after the live AUR network query, merged and re-scored.
A new search cancels the previous search's in-flight AUR request (Go context cancellation).

## Repo layout

```
backend/                  Go sidecar (module sword/backend)
  main.go                 IPC read/write loop
  models/                 shared types (SourcePackage, AppEntry, ...)
  sources/                Source interface + pacman / aur / flatpak impls
  metadata/               AppStream parsing + icon resolution
  registry/               in-memory AppIndex, dedup/merge
  search/                 two-phase orchestrator + scoring
frontend/
  src/ipc/backend.ts      sidecar lifecycle + IPC correlation
  src/api/                searchApps / fetchApps (thin wrappers over ipc)
  src/screens/SearchScreen.tsx
  src/components/ui/AppCard.tsx
  src-tauri/              Rust shell
    binaries/             sidecar binary lives here (gitignored build output)
    capabilities/default.json   shell + asset permissions
    tauri.conf.json       externalBin + assetProtocol config
```

## Source extensibility

Adding a package source = one new file implementing `sources.Source`
(`Name`, `Search`, `Get`, `Install`) + one registration line in `backend/main.go`.
This is the only intended extension point.

## Dev commands

The sidecar binary must exist **before** running the app. Tauri looks for it at
`frontend/src-tauri/binaries/sword-backend-<target-triple>`.

Get the target triple: `rustc -vV | grep host` (here: `x86_64-unknown-linux-gnu`).

**Build the Go sidecar** (re-run after any Go change):
```sh
cd backend
go build -o ../frontend/src-tauri/binaries/sword-backend-x86_64-unknown-linux-gnu .
```

**Run the whole app** (Vite + Rust + sidecar):
```sh
cd frontend
npm install            # first time only
npm run tauri dev
```

**Build / check individual layers**:
```sh
cd backend            && go build ./... && go vet ./...
cd frontend           && npx tsc --noEmit && npm run build
cd frontend/src-tauri && cargo check
```

**Test the Go backend in isolation** (no GUI). It reads JSON lines from stdin.
The index needs ~20-40s to build (flatpak listing + Flathub feed), so wait before querying:
```sh
( sleep 40; printf '{"type":"search","id":"r1","query":"firefox"}\n'; sleep 8 ) \
  | ./frontend/src-tauri/binaries/sword-backend-x86_64-unknown-linux-gnu
```
Search results print to stdout; progress/errors print to stderr.

## Gotchas

- **Rebuild the sidecar after editing Go code.** `tauri dev` does not do it for you.
- `expac` (from the `expac` package) must be installed for pacman results. Without it the
  pacman source fails gracefully and is excluded — no crash.
- The Arch distro AppStream feed has no canonical URL; `DistroFeedResolver` reads
  `SWORD_DISTRO_FEED_URL` and is a no-op when unset. Flathub feed is always on.
- The index is in-memory only, rebuilt every 30 min. No database, no disk cache.
- Icons: local metainfo -> distro feed -> Flathub feed -> "" (frontend placeholder).
  Local `file://` icons are routed through Tauri's asset protocol in `ipc/backend.ts`.
