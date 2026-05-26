import { create } from "zustand";
import { AppEntry } from "../types/app";

type UIStore = {
  theme: "dark" | "light";
  toggleTheme: () => void;
  activePanel: string;
  setActivePanel: (panel: string) => void;
  sourceOverrides: Record<string, string>;
  setSourceOverride: (appId: string, sourceId: string) => void;
  updatesAvailable: string[];
  setUpdatesAvailable: (pkgs: string[]) => void;
  currentInstall: string | null;
  setCurrentInstall: (pkg: string | null) => void;
  activeAppId: string | null;
  activeAppEntry: AppEntry | null;
  viewApp: (entry: AppEntry) => void;
  closeApp: () => void;
};

export const useUIStore = create<UIStore>((set, get) => ({
  theme: "dark",
  activePanel: "home",
  setActivePanel: (panel) => set({ activePanel: panel, activeAppId: null, activeAppEntry: null }),
  toggleTheme: () => {
    const next = get().theme === "dark" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", next);
    document.documentElement.className = next;
    set({ theme: next });
  },
  sourceOverrides: {},
  setSourceOverride: (appId, sourceId) =>
    set((s) => ({ sourceOverrides: { ...s.sourceOverrides, [appId]: sourceId } })),
  updatesAvailable: [],
  setUpdatesAvailable: (pkgs) => set({ updatesAvailable: pkgs }),
  currentInstall: null,
  setCurrentInstall: (pkg) => set({ currentInstall: pkg }),
  activeAppId: null,
  activeAppEntry: null,
  viewApp: (entry) => set({ activeAppId: entry.id, activeAppEntry: entry }),
  closeApp: () => set({ activeAppId: null, activeAppEntry: null }),
}));
