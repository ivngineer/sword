<div align="center">
  <img src="https://github.com/user-attachments/assets/f26fa5dd-6fe7-479c-a59f-4e61d941bfb4" width="128" alt="Sword Logo" />
  <h1>sword</h1>
  <p>A package manager for Linux. The goal is to make software management as easy and straightforward as in mobile operating systems.</p>
  <p>Sword stands for <strong>System Wide Open Repository Director</strong>.</p>
</div>

<div align="center">
  <img width="49%" src="https://github.com/user-attachments/assets/219856e5-75af-46f0-8e7f-2ce2e101fbb5" alt="2026-05-23_12-33-11" />
  <img width="49%" src="https://github.com/user-attachments/assets/219856e5-75af-46f0-8e7f-2ce2e101fbb5" alt="2026-05-23_12-33-11" />
</div>

## Status

Currently the app is WIP (work in progress) and comes with no promises.
Here's what works:

- Homescreen with most popular apps
- App cards showing name, description, icon, and active source
- Search engine across Pacman, Flatpak and AUR with deduplication (1 app = 1 entry)
- Multi-source unification: one entry per app, best source pre-selected, manual override available
- Dark and light theme with live switching
- Install and remove packages (may be unstable)
- App detail view with size, sources, and screenshots

## Known issues:

- Flatpak installs sometimes don't work
- Some icons in menus fail to render - expected to be fixed soon

## Dependencies

Runtime tools Sword shells out to. Missing ones degrade gracefully (that source is skipped, no crash).



|Tool     |Package  |
|---------|---------|
|`expac`  |`expac`  |
|`pacman` |`pacman` |
|`flatpak`|`flatpak`|
|`pkexec` |`polkit` |

Install everything in one go:

sudo pacman -S expac flatpak polkit


## Install and remove notes

- Pacman installs use pkexec and your desktop's polkit agent for authentication. The default agent is hyprpolkit. A custom built-in auth dialog is planned for a future release.
- Flatpak installs are per-user (flatpak install --user) - no password prompt needed, but installed apps won't be visible to other users.
- AUR installs require paru or yay on PATH. If AUR installs fail, set SUDO_ASKPASS to a graphical sudo prompt like ksshaskpass so sudo doesn't need a terminal.


## Near-term priorities:

- Installed apps list: separate view for what's currently on the system
- Update queue: pending updates across all sources in one place (including system packages!)
- Custom polkit agent: native, theme-matching auth dialog built into Sword
- Smoothness optimizations: it runs well on my machine, but I don't think it would on a 2010 laptop

Built on Tauri with Go on-device backend.
May be bloated; I prioritize UX over sparing 200mb of ram.
