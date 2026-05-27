import { Topbar } from "../components/layout/Topbar";
import { AppGrid } from "../components/layout/AppGrid";
import { fetchInstalledApps } from "../api/apps";

export function InstalledScreen() {
  return (
    <>
      <Topbar />
      <AppGrid
        queryKey={["installed"]}
        fetchFn={fetchInstalledApps}
        emptyMessage="No installed apps yet"
      />
    </>
  );
}
