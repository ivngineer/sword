import { ReactNode, useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@heroui/react";
import { AppCard } from "../ui/AppCard";
import { fetchPopularApps } from "../../api/apps";
import { AppEntry } from "../../types/app";

const MAX_POPULAR_POLLS = 12; // 12 × 5s = 60s, covers 20-40s index build + refreshPopular
const PAGE_SIZE = 30;

type AppGridProps = {
  queryKey?: readonly unknown[];
  fetchFn?: () => Promise<AppEntry[]>;
  emptyMessage?: ReactNode;
};

export function AppGrid({
  queryKey = ["popular"],
  fetchFn = fetchPopularApps,
  emptyMessage = "No popular apps available yet",
}: AppGridProps = {}) {
  const pollCount = useRef(0);
  const isPopular = queryKey[0] === "popular";

  const { data: apps = [], isLoading, isFetching } = useQuery({
    queryKey,
    queryFn: async () => {
      pollCount.current += 1;
      return fetchFn();
    },
    staleTime: 30 * 60 * 1000,
    refetchOnWindowFocus: false,
    refetchInterval: (query) => {
      if (!isPopular) return false;
      const apps = query.state.data ?? [];
      const hasResults = apps.length > 0;
      const allHaveIcons = hasResults && apps.every((a) => a.iconUrl);
      if (allHaveIcons) return false;
      if (pollCount.current >= MAX_POPULAR_POLLS) return false;
      return 5_000;
    },
  });

  const stillPolling = isPopular && (isFetching || pollCount.current < MAX_POPULAR_POLLS);

  // Incremental render: only mount the first N cards, grow as the user scrolls
  // near the bottom. Keeps long lists (Installed: ~100s) cheap to scroll on a
  // first paint and avoids paying for every off-screen AppCard upfront.
  const [visible, setVisible] = useState(PAGE_SIZE);
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    setVisible(PAGE_SIZE);
  }, [queryKey.join("|"), apps.length]);

  useEffect(() => {
    const node = sentinelRef.current;
    if (!node) return;
    if (visible >= apps.length) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          setVisible((v) => Math.min(v + PAGE_SIZE, apps.length));
        }
      },
      { rootMargin: "400px 0px" },
    );
    io.observe(node);
    return () => io.disconnect();
  }, [visible, apps.length]);

  if (isLoading || (apps.length === 0 && stillPolling)) {
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
          {emptyMessage}
        </p>
      </div>
    );
  }

  const shown = apps.slice(0, visible);
  return (
    <>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        {shown.map((entry) => (
          <AppCard key={entry.id} entry={entry} />
        ))}
      </div>
      {visible < apps.length && <div ref={sentinelRef} className="h-1" />}
    </>
  );
}
