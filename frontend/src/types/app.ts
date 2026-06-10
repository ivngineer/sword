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
