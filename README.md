<div align="center">
  <img src="https://github.com/user-attachments/assets/f26fa5dd-6fe7-479c-a59f-4e61d941bfb4" width="128" alt="Sword Logo" />
  <h1>sword</h1>
  <p>A package manager for Linux. The goal is to make software management as easy and straightforward as in mobile operating systems.</p>
  <p>Sword stands for <strong>System Wide Open Repository Director</strong>.</p>
</div>

<div align="center">
  <img width="49%" src="https://github.com/user-attachments/assets/832c1e77-fb4e-4bc5-b77d-6aaf7c26b724" alt="2026-05-23_12-33-11" />
  <img width="49%" src="https://github.com/user-attachments/assets/6ac72f31-9077-47be-93f3-d8918e8cd6b3" alt="2026-05-23_12-33-43" />
</div>

## Status
Currently the app is WIP (work in progress) and comes with no promises. 
Here's what works:
- Homescreen with most popular apps
- App cards showing name, description, icon, and active source
- Search engine across Pacman, Flatpak and AUR with deduplication (1 app = 1 entry)
- Multi-source unification: one entry per app, best source pre-selected, manual override available
- Dark and light theme with live switching

## Dependencies
Runtime tools Sword shells out to. Missing ones degrade gracefully (that source is skipped, no crash).

| Tool | Package | Purpose |
| --- | --- | --- |
| `expac` | `expac` | Query Arch sync databases for pacman search results. Without it, **no pacman packages appear**. |
| `pacman` | `pacman` | Detect installed native and AUR (foreign) packages via `-Qqn` / `-Qqm`. Preinstalled on Arch. |
| `flatpak` | `flatpak` | List Flathub apps and detect installed flatpaks. Without it, the Flatpak source is disabled. |
| `pkexec` | `polkit` | Elevate privileges for `pacman -S` installs. |

Install everything in one go:
```sh
sudo pacman -S expac flatpak polkit
```

The AUR source uses the AUR RPC over HTTPS — no local helper required.

### Install / remove caveats

- **A graphical polkit agent must be running** for pacman installs. Most desktop
  environments (GNOME, KDE, Xfce, LXQt, MATE) ship one and autostart it. On
  minimal window managers (i3, sway, dwm, etc.) install one — for example
  `polkit-gnome` — and ensure it autostarts with your session:
  ```sh
  sudo pacman -S polkit-gnome
  # then in your WM autostart:
  /usr/lib/polkit-gnome/polkit-gnome-authentication-agent-1 &
  ```
  Sword will try to spawn an agent itself if it finds one of the common
  binaries on PATH and none is running, but only as a fallback.
- **Flatpak installs are per-user** (`flatpak install --user`). This avoids a
  polkit prompt entirely; the trade-off is that flatpaks installed via Sword
  won't be visible to other users.
- **AUR installs require `paru` or `yay`** on PATH. The helper's internal
  `sudo` call may still want a TTY when invoked from a GUI; if you hit an
  AUR install failure, configure `SUDO_ASKPASS` (e.g. point it at
  `ksshaskpass` or `lxqt-openssh-askpass`) so sudo can prompt graphically.
- **No custom auth dialog yet.** The password prompt is rendered by your
  desktop's polkit agent, not by Sword. A native, theme-matching auth dialog
  would require implementing the polkit `AuthenticationAgent` D-Bus
  interface — on the roadmap, not in this release.

## Roadmap
Near-term priorities:
- **Install and remove**: one-click package management functionality
- **App detail view**: full description, version history, source comparison, screenshots
- **Installed apps list**: separate view for what's currently on the system
- **Update queue**: pending updates across all sources in one place (including system packages!)
- **Smoothness optimizations**: it runs well on my machine, but I don't think it would on a 2010 laptop

Built on Tauri with Go on-device backend. 
May be bloated; I prioritize UX over sparing 200mb of ram.
