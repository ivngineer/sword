<img src="https://github.com/user-attachments/assets/f26fa5dd-6fe7-479c-a59f-4e61d941bfb4" width="128" alt="Sword Logo" />

# Sword
A package manager for Linux. The goal is to make software management as easy and straightforward as in mobile operating systems.

Sword stands for **System Wide Open Repository Director**.

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
