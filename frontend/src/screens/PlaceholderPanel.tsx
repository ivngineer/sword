export function PlaceholderPanel({ name }: { name: string }) {
  return (
    <div className="flex flex-1 items-center justify-center flex-col gap-3">
      <span
        className="text-4xl font-semibold capitalize"
        style={{ color: "var(--foreground-muted, var(--foreground))", opacity: 0.2 }}
      >
        {name}
      </span>
      <span
        className="text-sm"
        style={{ color: "var(--foreground-muted, var(--foreground))", opacity: 0.15 }}
      >
        Coming soon
      </span>
    </div>
  );
}
