import { useState } from "react";
import { Button } from "@heroui/react";
import { AppEntry } from "../../types/app";
import { SourceSwitcher } from "./SourceSwitcher";

export function AppCard({ entry }: { entry: AppEntry }) {
  const [imgError, setImgError] = useState(false);

  return (
    <div
      className="rounded-xl p-5 h-[210px]"
      style={{ backgroundColor: "var(--surface-secondary)" }}
    >
      <div className="flex flex-row gap-5 items-start h-full">
        {/* Icon block */}
        <div
          className="w-[117px] h-[117px] shrink-0 flex items-center justify-center"
        >
          {imgError ? (
            <span className="text-sm" style={{ color: "var(--muted)" }}>
              Icon
            </span>
          ) : (
            <img
              src={entry.iconUrl}
              className="w-full h-full object-contain"
              alt={entry.name}
              onError={() => setImgError(true)}
            />
          )}
        </div>

        {/* Text + action stack */}
        <div className="flex flex-col justify-between flex-1 h-full min-w-0">
          <div>
            <h3
              className="text-[22px] font-semibold leading-tight truncate"
              style={{ color: "var(--foreground)" }}
            >
              {entry.name}
            </h3>
            <p
              className="text-sm mt-1 line-clamp-2"
              style={{ color: "var(--muted)" }}
            >
              {entry.description}
            </p>
          </div>

          <div className="flex items-center gap-3">
            {entry.sources.length > 1 && <SourceSwitcher entry={entry} />}
            <Button
              variant="secondary"
              size="sm"
              className="rounded-full px-5"
              style={{
                backgroundColor: "var(--surface-tertiary)",
                color: "var(--foreground)",
              }}
            >
              Get
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
