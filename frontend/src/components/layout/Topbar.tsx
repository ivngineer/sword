import { ThemeToggle } from "../ui/ThemeToggle";

export function Topbar() {
  return (
    <div
      className="w-full h-[68px] rounded-xl flex items-center justify-between px-6 mb-6 shrink-0"
      style={{ backgroundColor: "var(--surface)" }}
    >
      <p className="text-sm truncate" style={{ color: "var(--foreground)" }}>
        8 updates: steam, gimp, linux, sudo, …
      </p>

      <div className="flex items-center gap-3">
        {/* Indeterminate spinner — CSS fallback since CircularProgress v3 docs were unavailable */}
        <svg
          className="animate-spin"
          width="20"
          height="20"
          viewBox="0 0 20 20"
          fill="none"
          aria-label="Installing"
        >
          <circle
            cx="10"
            cy="10"
            r="8"
            stroke="var(--muted)"
            strokeWidth="2"
            strokeDasharray="40"
            strokeDashoffset="10"
            strokeLinecap="round"
          />
        </svg>
        <span className="text-sm" style={{ color: "var(--foreground)" }}>
          current install: Firefox
        </span>
        <ThemeToggle />
      </div>
    </div>
  );
}
