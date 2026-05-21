import { AppEntry } from "../types/app";

export const MOCK_APPS: AppEntry[] = [
  {
    id: "inkscape",
    name: "Inkscape",
    publisher: "Inkscape Project",
    description: "Professional vector graphics editor. Create illustrations, logos, diagrams, and more.",
    iconUrl: "https://dl.flathub.org/repo/appstream/x86_64/icons/128x128/org.inkscape.Inkscape.png",
    sources: [
      {
        id: "inkscape-pacman",
        type: "pacman",
        packageName: "inkscape",
        version: "1.3.2-1",
        sizeBytes: 95_000_000,
        isRecommended: true,
      },
      {
        id: "inkscape-flatpak",
        type: "flatpak",
        packageName: "org.inkscape.Inkscape",
        version: "1.3.2",
        sizeBytes: 220_000_000,
        isRecommended: false,
      },
    ],
  },
  {
    id: "gimp",
    name: "GIMP",
    publisher: "GNOME Project",
    description: "GNU Image Manipulation Program. A free and open-source raster graphics editor.",
    iconUrl: "https://dl.flathub.org/repo/appstream/x86_64/icons/128x128/org.gimp.GIMP.png",
    sources: [
      {
        id: "gimp-pacman",
        type: "pacman",
        packageName: "gimp",
        version: "2.10.36-2",
        sizeBytes: 85_000_000,
        isRecommended: true,
      },
      {
        id: "gimp-flatpak",
        type: "flatpak",
        packageName: "org.gimp.GIMP",
        version: "2.10.36",
        sizeBytes: 260_000_000,
        isRecommended: false,
      },
    ],
  },
  {
    id: "obs",
    name: "OBS Studio",
    publisher: "OBS Project",
    description: "Free and open-source software for video recording and live streaming.",
    iconUrl: "https://dl.flathub.org/repo/appstream/x86_64/icons/128x128/com.obsproject.Studio.png",
    sources: [
      {
        id: "obs-pacman",
        type: "pacman",
        packageName: "obs-studio",
        version: "30.2.2-1",
        sizeBytes: 48_000_000,
        isRecommended: true,
      },
    ],
  },
  {
    id: "steam",
    name: "Steam",
    publisher: "Valve Corporation",
    description: "Digital distribution platform for video games. Access your game library anywhere.",
    iconUrl: "https://dl.flathub.org/repo/appstream/x86_64/icons/128x128/com.valvesoftware.Steam.png",
    sources: [
      {
        id: "steam-flatpak",
        type: "flatpak",
        packageName: "com.valvesoftware.Steam",
        version: "1.0.0.81",
        sizeBytes: 310_000_000,
        isRecommended: true,
      },
    ],
  },
  {
    id: "boxes",
    name: "GNOME Boxes",
    publisher: "GNOME Project",
    description: "Simple virtualization tool for running virtual machines and remote desktops.",
    iconUrl: "https://dl.flathub.org/repo/appstream/x86_64/icons/128x128/org.gnome.Boxes.png",
    sources: [
      {
        id: "boxes-flatpak",
        type: "flatpak",
        packageName: "org.gnome.Boxes",
        version: "46.1",
        sizeBytes: 72_000_000,
        isRecommended: true,
      },
    ],
  },
  {
    id: "yay",
    name: "Yay",
    publisher: "Jguer",
    description: "Yet Another Yogurt — AUR helper and pacman wrapper with extended features.",
    iconUrl: "https://dl.flathub.org/repo/appstream/x86_64/icons/128x128/org.gnome.Boxes.png",
    sources: [
      {
        id: "yay-aur",
        type: "aur",
        packageName: "yay",
        version: "12.3.5-1",
        sizeBytes: 4_200_000,
        isRecommended: true,
      },
    ],
  },
];
