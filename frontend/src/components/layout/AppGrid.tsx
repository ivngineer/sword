import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@heroui/react";
import { AppCard } from "../ui/AppCard";
import { fetchPopularApps } from "../../api/apps";

export function AppGrid() {
  const { data: apps = [], isLoading } = useQuery({
    queryKey: ["popular"],
    queryFn: fetchPopularApps,
    staleTime: 30 * 60 * 1000,
    refetchOnWindowFocus: false,
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
