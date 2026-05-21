import { create } from "zustand";

type UIStore = {
  theme: "dark" | "light";
  toggleTheme: () => void;
  sourceOverrides: Record<string, string>;
  setSourceOverride: (appId: string, sourceId: string) => void;
};

export const useUIStore = create<UIStore>((set, get) => ({
  theme: "dark",
  toggleTheme: () => {
    const next = get().theme === "dark" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", next);
    document.documentElement.className = next;
    set({ theme: next });
  },
  sourceOverrides: {},
  setSourceOverride: (appId, sourceId) =>
    set((s) => ({ sourceOverrides: { ...s.sourceOverrides, [appId]: sourceId } })),
}));
