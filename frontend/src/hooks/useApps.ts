import { useQuery } from "@tanstack/react-query";
import { fetchApps } from "../api/apps";
import { AppEntry } from "../types/app";

type AppsQuery = { q?: string; limit?: number; offset?: number };

export function useApps(query: AppsQuery): {
  apps: AppEntry[];
  isLoading: boolean;
  isError: boolean;
} {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["apps", query],
    queryFn: () => fetchApps(query),
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  });

  return { apps: data?.apps ?? [], isLoading, isError };
}
