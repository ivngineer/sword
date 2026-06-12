import { useCallback, useEffect, useState } from "react";
import { AppGrid } from "../components/layout/AppGrid";
import { FirmwareBanner } from "../components/ui/FirmwareBanner";
import { fetchDriversPhased, installFirmwarePackages, scanFirmware } from "../api/apps";
import { AppEntry, FirmwarePackage } from "../types/app";

// DriversScreen streams driver results: the instant pacman scan renders first,
// then AUR results are appended when they arrive (progressive load). Opening
// the tab also kicks off an async firmware scan; missing firmware packages
// surface in the FirmwareBanner above the grid (hidden when none).
export function DriversScreen() {
  const [apps, setApps] = useState<AppEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [firmware, setFirmware] = useState<FirmwarePackage[]>([]);
  // Pending install list (package names); the review dialog edits this.
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [installing, setInstalling] = useState(false);

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
    scanFirmware()
      .then((pkgs) => {
        if (!alive) return;
        setFirmware(pkgs);
        setSelected(new Set(pkgs.map((p) => p.name)));
      })
      .catch((err) => console.error("[firmware]", err));
    return () => {
      alive = false;
    };
  }, []);

  const installFirmware = useCallback(async () => {
    const names = firmware.map((p) => p.name).filter((n) => selected.has(n));
    if (names.length === 0) return;
    setInstalling(true);
    try {
      await installFirmwarePackages(names);
      // Re-scan so the panel reflects (and usually hides after) the install.
      const pkgs = await scanFirmware();
      setFirmware(pkgs);
      setSelected(new Set(pkgs.map((p) => p.name)));
    } catch (err) {
      console.error("[firmware] install:", err);
    } finally {
      setInstalling(false);
    }
  }, [firmware, selected]);

  return (
    <>
      {firmware.length > 0 && (
        <FirmwareBanner
          packages={firmware}
          selected={selected}
          onSelectedChange={setSelected}
          onInstall={installFirmware}
          installing={installing}
        />
      )}
      <AppGrid apps={apps} loading={loading} emptyMessage="No drivers found" />
    </>
  );
}
