import { Sidebar } from "../components/layout/Sidebar";
import { Topbar } from "../components/layout/Topbar";
import { AppGrid } from "../components/layout/AppGrid";
import { PlaceholderPanel } from "./PlaceholderPanel";
import { AboutScreen } from "./AboutScreen";
import { SettingsScreen } from "./SettingsScreen";
import { SearchScreen } from "./SearchScreen";
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

  return (
    <div className="flex h-screen w-screen overflow-hidden">
      <Sidebar />
      <main
        className={`flex flex-1 flex-col overflow-y-auto ${activePanel === "search" ? "" : "p-6"}`}
        style={{ backgroundColor: "var(--background)" }}
      >
        {activePanel === "home" ? (
          <HomePanel />
        ) : activePanel === "search" ? (
          <SearchScreen />
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
