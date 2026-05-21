import { useState, useRef, useCallback, useEffect } from "react";
import { InputGroup, Skeleton, Card } from "@heroui/react";
import { Search } from "lucide-react";
import { AppCard } from "../components/ui/AppCard";
import { searchApps, SearchResult } from "../api/search";
import { tokens } from "../theme/tokens";

type Phase = "idle" | "loading" | "results";

// Sits above true center — feels more natural for a search entry point
const IDLE_TOP = "38%";
const INPUT_HEIGHT = "44px";

export function SearchScreen() {
  const [query, setQuery] = useState("");
  const [phase, setPhase] = useState<Phase>("idle");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [total, setTotal] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const runSearch = useCallback(async (q: string) => {
    abortRef.current?.abort();
    abortRef.current = new AbortController();
    setPhase("loading");
    setResults([]);
    try {
      const res = await searchApps(q);
      setResults(res.results);
      setTotal(res.total);
      setPhase("results");
    } catch {
      setPhase("results");
    }
  }, []);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" && query.trim()) runSearch(query.trim());
    if (e.key === "Escape") clear();
  };

  const clear = () => {
    abortRef.current?.abort();
    setQuery("");
    setPhase("idle");
    setResults([]);
    inputRef.current?.focus();
  };

  const hasQuery = phase !== "idle";

  return (
    <div className="relative flex flex-col w-full h-full overflow-hidden">
      {/* Search bar — slides from ~38% to top on search */}
      <div
        className="absolute left-0 right-0 z-10"
        style={{
          top: hasQuery ? "0" : IDLE_TOP,
          transform: hasQuery ? "translateY(0)" : "translateY(-50%)",
          padding: hasQuery
            ? `${tokens.spacing.outer} ${tokens.spacing.outer} 0`
            : `0 ${tokens.spacing.outer}`,
          transition: [
            "top 0.45s cubic-bezier(0.4, 0, 0.2, 1)",
            "transform 0.45s cubic-bezier(0.4, 0, 0.2, 1)",
            "padding 0.45s cubic-bezier(0.4, 0, 0.2, 1)",
          ].join(", "),
        }}
      >
        {!hasQuery && (
          <p
            className="text-center text-4xl font-semibold mb-5 select-none capitalize"
            style={{ color: "var(--foreground-muted, var(--foreground))", opacity: 0.2 }}
          >
            Find apps
          </p>
        )}

        <div className="flex justify-center">
        <InputGroup
          variant="secondary"
          style={{ height: INPUT_HEIGHT, width: "75%" }}
        >
          <InputGroup.Prefix>
            <Search size={15} style={{ color: "var(--muted)" }} />
          </InputGroup.Prefix>

          <InputGroup.Input
            ref={inputRef}
            placeholder="Search apps…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
          />

          {query && (
            <InputGroup.Suffix>
              <button
                onClick={clear}
                className="text-xs rounded px-1"
                style={{ color: "var(--muted)" }}
                tabIndex={-1}
                aria-label="Clear"
              >
                esc
              </button>
            </InputGroup.Suffix>
          )}
        </InputGroup>
        </div>
      </div>

      {/* Results list */}
      <div
        className="flex-1 overflow-y-auto"
        style={{
          opacity: hasQuery ? 1 : 0,
          transition: "opacity 0.3s ease 0.25s",
          paddingTop: hasQuery
            ? `calc(${tokens.spacing.outer} + ${INPUT_HEIGHT} + ${tokens.spacing.outer})`
            : 0,
          paddingInline: tokens.spacing.outer,
          paddingBottom: tokens.spacing.outer,
          pointerEvents: hasQuery ? "auto" : "none",
        }}
      >
        {phase === "loading" && (
          <div className="flex flex-col gap-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <SkeletonCard key={i} />
            ))}
          </div>
        )}

        {phase === "results" && results.length === 0 && (
          <p
            className="text-center mt-16 text-sm"
            style={{ color: "var(--muted)" }}
          >
            No results for &ldquo;{query}&rdquo;
          </p>
        )}

        {phase === "results" && results.length > 0 && (
          <>
            <p className="text-xs mb-3" style={{ color: "var(--muted)" }}>
              {total} result{total !== 1 ? "s" : ""}
            </p>
            <div className="flex flex-col gap-3">
              {results.map((r, i) => (
                <div
                  key={r.id}
                  style={{
                    animation: "fadeSlideIn 0.3s ease both",
                    animationDelay: `${i * 50}ms`,
                  }}
                >
                  <AppCard entry={r} />
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      <style>{`
        @keyframes fadeSlideIn {
          from { opacity: 0; transform: translateY(8px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  );
}

function SkeletonCard() {
  return (
    <Card className="h-[189px]">
      <Card.Content className="h-full flex flex-row gap-5 p-5 items-start">
        <Skeleton className="w-[100px] h-[100px] rounded-xl shrink-0" />
        <div className="flex flex-col gap-2.5 flex-1 pt-1">
          <Skeleton className="h-[22px] w-2/5 rounded-lg" />
          <Skeleton className="h-3.5 w-full rounded" />
          <Skeleton className="h-3.5 w-3/4 rounded" />
        </div>
      </Card.Content>
    </Card>
  );
}
