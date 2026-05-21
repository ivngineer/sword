import { Button } from "@heroui/react";
import { Sun, Moon } from "lucide-react";
import { useUIStore } from "../../store/ui.store";

export function ThemeToggle() {
  const { theme, toggleTheme } = useUIStore();
  return (
    <Button
      variant="ghost"
      size="sm"
      isIconOnly
      onPress={toggleTheme}
      aria-label="Toggle theme"
    >
      {theme === "dark" ? <Sun size={16} /> : <Moon size={16} />}
    </Button>
  );
}
