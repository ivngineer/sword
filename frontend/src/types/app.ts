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

export type AppEntry = {
  id: string;
  name: string;
  publisher: string;
  description: string;
  iconUrl: string;
  status: AppStatus;
  sources: AppSource[];
  screenshots?: string[];
};
