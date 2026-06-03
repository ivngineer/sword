import { useEffect, useState } from "react";
import { Topbar } from "../components/layout/Topbar";
import { AppGrid } from "../components/layout/AppGrid";
import { fetchDriversPhased } from "../api/apps";
import { AppEntry } from "../types/app";

// DriversScreen streams driver results: the instant pacman scan renders first,
// then AUR results are appended when they arrive (progressive load).
export function DriversScreen() {
  const [apps, setApps] = useState<AppEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    fetchDriversPhased((_phase, results) => {
      if (alive) {
        setApps(results);
        setLoading(false);
      }
    }).catch((err) => {
      console.error("[drivers]", err);
      if (alive) setLoading(false);
    });
    return () => {
      alive = false;
    };
  }, []);

  return (
    <>
      <Topbar />
      <AppGrid apps={apps} loading={loading} emptyMessage="No drivers found" />
    </>
  );
}
