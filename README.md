<div align="center">
  <img src="https://github.com/user-attachments/assets/f26fa5dd-6fe7-479c-a59f-4e61d941bfb4" width="128" alt="Sword Logo" />

  # sword
</div>

A package manager for Linux. The goal is to make software management as easy and straightforward as in mobile operating systems.

Sword stands for **System Wide Open Repository Director**.


## Status
Currently the app is WIP (work in progress) and comes with no promises. 
Here's what works:
- Homescreen with most popular apps
- App cards showing name, description, icon, and active source
- Search engine across Pacman, Flatpak and AUR with deduplication (1 app = 1 entry)
- Multi-source unification: one entry per app, best source pre-selected, manual override available
- Dark and light theme with live switching

## Roadmap
Near-term priorities:
- **Install and remove**: one-click package management functionality
- **App detail view**: full description, version history, source comparison, screenshots
- **Installed apps list**: separate view for what's currently on the system
- **Update queue**: pending updates across all sources in one place (including system packages!)
- **Smoothness optimizations**: it runs well on my machine, but I don't think it would on a 2010 laptop

Built on Tauri with Go on-device backend. 
May be bloated; I prioritize UX over sparing 200mb of ram.
