import { useState } from "react";
import { Button } from "@heroui/react";
import { Check, Cpu } from "lucide-react";
import { FirmwarePackage } from "../../types/app";
import { formatBytes } from "../../lib/format";

type FirmwareBannerProps = {
  packages: FirmwarePackage[];
  // Pending install list (package names). Edited via the review dialog.
  selected: Set<string>;
  onSelectedChange: (next: Set<string>) => void;
  onInstall: () => void;
  installing: boolean;
};

// FirmwareBanner is the featured panel at the top of the Drivers screen. It
// spans the full grid width (3 cards + 2 gaps) and matches AppCard height:
// 189px card = 149px icon + 2 × 20px padding.
export function FirmwareBanner({
  packages,
  selected,
  onSelectedChange,
  onInstall,
  installing,
}: FirmwareBannerProps) {
  const [reviewOpen, setReviewOpen] = useState(false);
  // Draft selection while the review dialog is open; committed on Confirm.
  const [draft, setDraft] = useState<Set<string>>(new Set());

  const openReview = () => {
    setDraft(new Set(selected));
    setReviewOpen(true);
  };

  const toggleDraft = (name: string) => {
    setDraft((d) => {
      const next = new Set(d);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const confirmReview = () => {
    onSelectedChange(draft);
    setReviewOpen(false);
  };

  return (
    <>
      <div
        className="rounded-xl p-5 mb-6 flex flex-row gap-5 items-center"
        style={{ backgroundColor: "var(--surface-secondary)" }}
      >
        {/* Large square firmware icon, card height minus padding */}
        <div
          className="w-[149px] h-[149px] shrink-0 rounded-lg flex items-center justify-center"
          style={{ backgroundColor: "var(--surface)" }}
        >
          <Cpu size={72} style={{ color: "var(--muted)" }} />
        </div>

        <div className="flex flex-col flex-1 min-w-0 gap-1">
          <h2
            className="text-[22px] font-semibold leading-tight select-none"
            style={{ color: "var(--foreground)" }}
          >
            New firmware available for your machine
          </h2>
          {/* Deselected packages stay visible but dimmed + struck through */}
          <p className="text-sm select-none" style={{ color: "var(--muted)" }}>
            {packages.map((p, i) => (
              <span key={p.name}>
                <span
                  style={
                    selected.has(p.name)
                      ? undefined
                      : { textDecoration: "line-through", opacity: 0.5 }
                  }
                >
                  {p.name}
                </span>
                {i < packages.length - 1 && ", "}
              </span>
            ))}
          </p>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <Button
            variant="secondary"
            size="sm"
            isDisabled={installing}
            onPress={openReview}
            className="rounded-full px-5"
            style={{ backgroundColor: "var(--surface)", color: "var(--foreground)" }}
          >
            Review Packages
          </Button>
          <Button
            variant="secondary"
            size="sm"
            isDisabled={installing || selected.size === 0}
            onPress={onInstall}
            className="rounded-full px-5"
            style={{
              backgroundColor: "#3b82f6",
              color: "#ffffff",
              opacity: installing || selected.size === 0 ? 0.6 : 1,
            }}
          >
            {installing ? "Installing…" : "Install All"}
          </Button>
        </div>
      </div>

      {reviewOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center"
          role="dialog"
          aria-modal="true"
          aria-label="Review firmware packages"
        >
          <div
            className="absolute inset-0 bg-black/60"
            onClick={() => setReviewOpen(false)}
          />
          <div
            className="relative w-[480px] max-h-[70vh] flex flex-col gap-4 rounded-xl p-5"
            style={{ backgroundColor: "var(--surface-secondary)" }}
          >
            <h3
              className="text-lg font-semibold select-none"
              style={{ color: "var(--foreground)" }}
            >
              Review firmware packages
            </h3>

            <div className="flex-1 min-h-0 overflow-y-auto flex flex-col gap-2">
              {packages.map((p) => {
                const on = draft.has(p.name);
                return (
                  <button
                    key={p.name}
                    type="button"
                    onClick={() => toggleDraft(p.name)}
                    className="flex items-start gap-3 rounded-lg p-3 text-left cursor-pointer"
                    style={{ backgroundColor: "var(--surface)" }}
                  >
                    <span
                      className="mt-0.5 w-5 h-5 shrink-0 rounded flex items-center justify-center border"
                      style={{
                        backgroundColor: on ? "#3b82f6" : "transparent",
                        borderColor: on ? "#3b82f6" : "var(--muted)",
                      }}
                    >
                      {on && <Check size={14} color="#ffffff" />}
                    </span>
                    <span className="flex flex-col min-w-0">
                      <span
                        className="text-sm font-medium select-none"
                        style={{ color: "var(--foreground)" }}
                      >
                        {p.name}
                        <span
                          className="ml-2 font-normal text-xs select-none"
                          style={{ color: "var(--muted)" }}
                        >
                          {p.version}
                          {formatBytes(p.sizeBytes) && ` · ${formatBytes(p.sizeBytes)}`}
                        </span>
                      </span>
                      {p.description && (
                        <span
                          className="text-xs line-clamp-2 select-none"
                          style={{ color: "var(--muted)" }}
                        >
                          {p.description}
                        </span>
                      )}
                    </span>
                  </button>
                );
              })}
            </div>

            <div className="flex items-center justify-end gap-3">
              <Button
                variant="secondary"
                size="sm"
                onPress={() => setReviewOpen(false)}
                className="rounded-full px-5"
                style={{ backgroundColor: "var(--surface)", color: "var(--foreground)" }}
              >
                Cancel
              </Button>
              <Button
                variant="secondary"
                size="sm"
                onPress={confirmReview}
                className="rounded-full px-5"
                style={{ backgroundColor: "#3b82f6", color: "#ffffff" }}
              >
                Confirm ({draft.size})
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
