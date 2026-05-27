import { Sidebar } from "../components/layout/Sidebar";
import { Topbar } from "../components/layout/Topbar";
import { AppGrid } from "../components/layout/AppGrid";
import { PlaceholderPanel } from "./PlaceholderPanel";
import { AboutScreen } from "./AboutScreen";
import { SettingsScreen } from "./SettingsScreen";
import { SearchScreen } from "./SearchScreen";
import { InstalledScreen } from "./InstalledScreen";
import { AppScreen } from "./AppScreen";
import { useUIStore } from "../store/ui.store";

function HomePanel() {
  return (
    <>
      <Topbar />
      <AppGrid />
    </>
  );
}

export function MainScreen() {
  const { activePanel } = useUIStore();
  const activeAppId = useUIStore((s) => s.activeAppId);
  const showApp = !!activeAppId;

  return (
    <div className="flex h-screen w-screen overflow-hidden">
      <Sidebar />
      <main
        className={`flex flex-1 flex-col overflow-y-auto ${showApp || activePanel === "search" ? "" : "p-6"}`}
        style={{ backgroundColor: "var(--background)" }}
      >
        {activePanel === "search" ? (
          <>
            <div
              className="flex flex-1 flex-col overflow-hidden"
              style={{ display: showApp ? "none" : "flex" }}
            >
              <SearchScreen />
            </div>
            {showApp && <AppScreen />}
          </>
        ) : showApp ? (
          <AppScreen />
        ) : activePanel === "home" ? (
          <HomePanel />
        ) : activePanel === "installed" ? (
          <InstalledScreen />
        ) : activePanel === "about" ? (
          <AboutScreen />
        ) : activePanel === "settings" ? (
          <SettingsScreen />
        ) : (
          <PlaceholderPanel name={activePanel} />
        )}
      </main>
    </div>
  );
}
