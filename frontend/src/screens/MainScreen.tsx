import { Sidebar } from "../components/layout/Sidebar";
import { Topbar } from "../components/layout/Topbar";
import { AppGrid } from "../components/layout/AppGrid";

export function MainScreen() {
  return (
    <div className="flex h-screen w-screen overflow-hidden">
      <Sidebar />
      <main
        className="flex flex-1 flex-col p-6 overflow-y-auto"
        style={{ backgroundColor: "var(--background)" }}
      >
        <Topbar />
        <AppGrid />
      </main>
    </div>
  );
}
