import { Select, ListBox } from "@heroui/react";
import { AppSource } from "../../types/app";

type Props = {
  sources: AppSource[];
  value: AppSource;
  onChange: (sourceId: string) => void;
};

export function SourceSwitcher({ sources, value, onChange }: Props) {
  if (sources.length === 0) return null;

  return (
    <Select
      variant="secondary"
      selectedKey={value.id}
      onSelectionChange={(key) => {
        if (key != null) onChange(String(key));
      }}
      className="min-w-0 flex-1"
      aria-label="Select source"
    >
      <Select.Trigger className="text-sm overflow-hidden rounded-full" style={{ color: "var(--foreground)" }}>
        <Select.Value className="truncate min-w-0" />
        <Select.Indicator className="shrink-0" />
      </Select.Trigger>
      <Select.Popover>
        <ListBox aria-label="Sources">
          {sources.map((src) => (
            <ListBox.Item
              key={src.id}
              id={src.id}
              textValue={`${src.type} ${src.version}`}
            >
              {src.type} {src.version}
            </ListBox.Item>
          ))}
        </ListBox>
      </Select.Popover>
    </Select>
  );
}
