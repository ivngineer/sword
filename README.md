# Sword

A package manager for Linux. The goal is to make installing and managing software feel as straightforward as it does on any modern desktop OS, without giving up the flexibility of the underlying package ecosystem.

Sword aims to make package management on Linux as user-friendly as possible.

---

## Status

The current version covers the main screen frontend:

- Two-pane layout with sidebar navigation and app grid
- App cards showing name, publisher, description, icon, and active source
- Multi-source unification: one entry per app, best source pre-selected, manual override available
- Dark and light theme with live switching
- Mock data layer standing in for the backend

Built with Tauri, React, TypeScript, and HeroUI v3.

---

## Roadmap

Near-term priorities:

- **Go backend** — local HTTP server querying pacman, AUR, and Flatpak, with deduplication and source ranking
- **Install and remove** — triggered via Tauri IPC, with real-time progress fed back to the UI
- **App detail view** — full description, version history, source comparison, size breakdown
- **Installed apps list** — separate view for what's currently on the system
- **Update queue** — pending updates across all sources in one place
- **Search and filter** — live search with source and category filters
