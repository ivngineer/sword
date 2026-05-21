import { ListBox } from "@heroui/react";
import { useUIStore } from "../../store/ui.store";
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
  Settings,
  RefreshCw,
  Info,
} from "lucide-react";

const TOP_ITEMS = [
  { id: "search", label: "Search", icon: Search },
  { id: "home", label: "Home", icon: Home },
  { id: "productivity", label: "Productivity", icon: Briefcase },
  { id: "development", label: "Development", icon: Code2 },
  { id: "art", label: "Art", icon: Palette },
  { id: "gaming", label: "Gaming", icon: Gamepad2 },
  { id: "graphics", label: "Graphics", icon: PenTool },
  { id: "communication", label: "Communication", icon: MessageSquare },
  { id: "utilities", label: "Utilities", icon: Wrench },
];

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
    </aside>
  );
}
