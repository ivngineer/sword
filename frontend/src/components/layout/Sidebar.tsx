import { useState } from "react";
import { ListBox, ProgressBar } from "@heroui/react";
import { useUIStore } from "../../store/ui.store";
import { useProgressStore, activeJob } from "../../store/progress.store";
import {
  Search,
  Home,
  Briefcase,
  Code2,
  Palette,
  Gamepad2,
  PenTool,
  MessageSquare,
  Wrench,
  Cpu,
  HardDrive,
  Settings,
  RefreshCw,
  Info,
  LayoutGrid,
  ChevronDown,
} from "lucide-react";

// Top nav split: fixed items above the Categories dropdown, fixed items below.
const TOP_ITEMS = [
  { id: "search", label: "Search", icon: Search },
  { id: "home", label: "Home", icon: Home },
];

// Program categories, collapsed under the "Categories" dropdown.
const CATEGORY_ITEMS = [
  { id: "productivity", label: "Productivity", icon: Briefcase },
  { id: "development", label: "Development", icon: Code2 },
  { id: "art", label: "Art", icon: Palette },
  { id: "gaming", label: "Gaming", icon: Gamepad2 },
  { id: "graphics", label: "Graphics", icon: PenTool },
  { id: "communication", label: "Communication", icon: MessageSquare },
  { id: "utilities", label: "Utilities", icon: Wrench },
];

const POST_ITEMS = [
  { id: "drivers", label: "Drivers", icon: Cpu },
  { id: "installed", label: "Installed", icon: HardDrive },
];

const CATEGORY_IDS = new Set(CATEGORY_ITEMS.map((c) => c.id));

const BOTTOM_ITEMS = [
  { id: "settings", label: "Settings", icon: Settings },
  { id: "updates", label: "Updates", icon: RefreshCw },
  { id: "about", label: "About", icon: Info },
];

function NavItem({
  id,
  label,
  Icon,
  isActive,
}: {
  id: string;
  label: string;
  Icon: React.ElementType;
  isActive: boolean;
}) {
  return (
    <ListBox.Item
      id={id}
      textValue={label}
      className="rounded-xl py-[10px] px-3 text-sm cursor-pointer select-none"
      style={{
        backgroundColor: isActive ? "var(--surface-secondary)" : "transparent",
        color: "var(--foreground)",
      }}
    >
      <div className="flex items-center gap-3">
        <Icon size={18} />
        <span>{label}</span>
      </div>
    </ListBox.Item>
  );
}

// CategoriesDropdown collapses the program categories into one expandable
// button that matches NavItem styling. Auto-opens when a category is active so
// the current selection is never hidden.
function CategoriesDropdown({
  activePanel,
  onSelect,
}: {
  activePanel: string;
  onSelect: (id: string) => void;
}) {
  const categoryActive = CATEGORY_IDS.has(activePanel);
  const [open, setOpen] = useState(categoryActive);

  return (
    <div className="flex flex-col gap-1">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="rounded-xl py-[10px] px-3 text-sm cursor-pointer select-none w-full"
        style={{
          backgroundColor:
            categoryActive && !open ? "var(--surface-secondary)" : "transparent",
          color: "var(--foreground)",
        }}
      >
        <div className="flex items-center gap-3">
          <LayoutGrid size={18} />
          <span>Categories</span>
          <ChevronDown
            size={16}
            className="ml-auto transition-transform"
            style={{ transform: open ? "rotate(180deg)" : "none" }}
          />
        </div>
      </button>

      {open && (
        <ListBox
          aria-label="Categories"
          selectionMode="single"
          selectedKeys={new Set([activePanel])}
          onSelectionChange={(keys) => {
            const k = [...keys][0];
            if (k) onSelect(String(k));
          }}
          className="flex flex-col gap-1 bg-transparent pl-3"
        >
          {CATEGORY_ITEMS.map(({ id, label, icon: Icon }) => (
            <NavItem key={id} id={id} label={label} Icon={Icon} isActive={activePanel === id} />
          ))}
        </ListBox>
      )}
    </div>
  );
}

export function Sidebar() {
  const { activePanel, setActivePanel } = useUIStore();

  const handleSelect = (keys: Iterable<unknown>) => {
    const k = [...keys][0];
    if (k) setActivePanel(String(k));
  };

  return (
    <aside
      className="w-[240px] h-full flex flex-col justify-between px-4 py-4"
      style={{ backgroundColor: "var(--surface)" }}
    >
      <div className="flex flex-col gap-1">
        <ListBox
          aria-label="Navigation"
          selectionMode="single"
          selectedKeys={new Set([activePanel])}
          onSelectionChange={handleSelect}
          className="flex flex-col gap-1 bg-transparent"
        >
          {TOP_ITEMS.map(({ id, label, icon: Icon }) => (
            <NavItem key={id} id={id} label={label} Icon={Icon} isActive={activePanel === id} />
          ))}
        </ListBox>

        <CategoriesDropdown activePanel={activePanel} onSelect={setActivePanel} />

        <ListBox
          aria-label="More navigation"
          selectionMode="single"
          selectedKeys={new Set([activePanel])}
          onSelectionChange={handleSelect}
          className="flex flex-col gap-1 bg-transparent"
        >
          {POST_ITEMS.map(({ id, label, icon: Icon }) => (
            <NavItem key={id} id={id} label={label} Icon={Icon} isActive={activePanel === id} />
          ))}
        </ListBox>
      </div>

      <div className="flex flex-col gap-2">
        <ActiveProgress />
        <ListBox
          aria-label="Bottom navigation"
          selectionMode="single"
          selectedKeys={new Set([activePanel])}
          onSelectionChange={handleSelect}
          className="flex flex-col gap-1 bg-transparent"
        >
          {BOTTOM_ITEMS.map(({ id, label, icon: Icon }) => (
            <NavItem key={id} id={id} label={label} Icon={Icon} isActive={activePanel === id} />
          ))}
        </ListBox>
      </div>
    </aside>
  );
}

// ActiveProgress renders a thin progress bar above the bottom nav whenever
// any install/remove is in flight. Switches between an accurate determinate
// bar and an indeterminate one based on what the backend can parse out of
// the tool's output.
function ActiveProgress() {
  const job = useProgressStore((s) => activeJob(s.jobs));
  if (!job) return null;
  const verb = job.kind === "install" ? "Installing" : "Removing";
  const pct =
    job.fraction == null ? null : Math.max(0, Math.min(100, Math.round(job.fraction * 100)));
  const isFlatpak = job.sourceType === "flatpak";
  return (
    <div
      className="mx-1 rounded-xl py-[10px] px-3 flex flex-col gap-2 select-none"
      style={{ backgroundColor: "var(--surface-secondary)" }}
    >
      <div className="flex items-center justify-between gap-2 text-sm">
        <span
          className="truncate"
          style={{ color: "var(--foreground)" }}
          title={isFlatpak ? `${verb} ${job.appName} · Flatpaks take longer to install` : `${verb} ${job.appName}`}
        >
          {verb} {job.appName}
        </span>
        {pct != null && (
          <span className="tabular-nums text-xs" style={{ color: "var(--muted)" }}>
            {pct}%
          </span>
        )}
      </div>
      <ProgressBar
        aria-label={`${verb} ${job.appName}`}
        value={pct ?? undefined}
        isIndeterminate={pct == null}
        size="sm"
        color="accent"
      >
        <ProgressBar.Track className="h-1.5 rounded-sm">
          <ProgressBar.Fill className="rounded-sm" />
        </ProgressBar.Track>
      </ProgressBar>
      {job.status && (
        <span
          className="truncate text-xs"
          style={{ color: "var(--muted)" }}
          title={job.status}
        >
          {job.status}
        </span>
      )}
    </div>
  );
}
