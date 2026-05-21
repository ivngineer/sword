export type AppSource = {
  id: string;
  type: "pacman" | "aur" | "flatpak";
  packageName: string;
  version: string;
  sizeBytes: number;
  isRecommended: boolean;
};

export type AppEntry = {
  id: string;
  name: string;
  publisher: string;
  description: string;
  iconUrl: string;
  sources: AppSource[];
};
