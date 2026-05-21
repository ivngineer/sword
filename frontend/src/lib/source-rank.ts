import { AppSource } from "../types/app";

const RANK: Record<AppSource["type"], number> = { pacman: 0, aur: 1, flatpak: 2 };

export function rankSources(sources: AppSource[]): AppSource {
  return [...sources].sort((a, b) => RANK[a.type] - RANK[b.type])[0];
}
