export type AppSource = {
  id: string;
  type: "pacman" | "aur" | "flatpak";
  packageName: string;
  version: string;
  sizeBytes: number;
  isRecommended: boolean;
  installed: boolean;
};

export type AppStatus = "available" | "installed";

// One missing firmware package recommended for the machine's hardware.
// name is the exact pacman package name used for installation.
export type FirmwarePackage = {
  name: string;
  description: string;
  version: string;
  sizeBytes: number;
};

// One pending package update. packageName is the pacman name or flatpak app
// id — the identifier the backend updates or skips. iconUrl is "" when the
// package owns no .desktop entry.
export type UpdateEntry = {
  name: string;
  packageName: string;
  sourceType: "pacman" | "aur" | "flatpak";
  iconUrl: string;
  currentVersion: string;
  newVersion: string;
};

// Per-manager ignore lists sent with a full system update.
export type UpdateSkip = {
  pacman: string[];
  aur: string[];
  flatpak: string[];
};

export type AppEntry = {
  id: string;
  name: string;
  publisher: string;
  description: string;
  iconUrl: string;
  status: AppStatus;
  sources: AppSource[];
  screenshots?: string[];
  driverKind?: "kernel-module" | "firmware" | "driver";
};
