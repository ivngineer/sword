import { Chip } from "@heroui/react";
import { AppSource } from "../../types/app";

const CONFIG: Record<
  AppSource["type"],
  { color: "default" | "warning" | "accent"; label: string }
> = {
  pacman: { color: "default", label: "pacman" },
  aur: { color: "warning", label: "AUR" },
  flatpak: { color: "accent", label: "Flatpak" },
};

export function SourceBadge({ type }: { type: AppSource["type"] }) {
  const { color, label } = CONFIG[type];
  return (
    <Chip size="sm" variant="soft" color={color}>
      {label}
    </Chip>
  );
}
