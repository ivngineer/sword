import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@heroui/react";
import { AppCard } from "../ui/AppCard";
import { fetchPopularApps } from "../../api/apps";

const MAX_POPULAR_POLLS = 12; // 12 × 5s = 60s, covers 20-40s index build + refreshPopular

export function AppGrid() {
  const pollCount = useRef(0);

  const { data: apps = [], isLoading } = useQuery({
    queryKey: ["popular"],
    queryFn: async () => {
      pollCount.current += 1;
      return fetchPopularApps();
    },
    staleTime: 30 * 60 * 1000,
    refetchOnWindowFocus: false,
    refetchInterval: (query) => {
      const apps = query.state.data ?? [];
      const hasResults = apps.length > 0;
      const allHaveIcons = hasResults && apps.every((a) => a.iconUrl);
      if (allHaveIcons) return false;
      if (pollCount.current >= MAX_POPULAR_POLLS) return false;
      return 5_000;
    },
  });

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="rounded-xl h-[189px]" />
        ))}
      </div>
    );
  }

  if (apps.length === 0) {
    return (
      <div className="flex items-center justify-center h-48">
        <p className="text-sm" style={{ color: "var(--muted)" }}>
          No popular apps available yet
        </p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
      {apps.map((entry) => (
        <AppCard key={entry.id} entry={entry} />
      ))}
    </div>
  );
}
