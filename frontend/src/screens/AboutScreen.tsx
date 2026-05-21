function LogoPlaceholder() {
  return (
    <img
      src="/logo.png"
      alt="Sword logo"
      style={{
        width: 96,
        height: 96,
        objectFit: "contain",
        flexShrink: 0,
      }}
    />
  );
}

export function AboutScreen() {
  return (
    <div
      className="flex flex-col items-center"
      style={{ height: "100%", paddingTop: "14vh", paddingBottom: "2.5rem" }}
    >
      <div
        className="flex flex-col items-center"
        style={{ gap: "1.25rem", textAlign: "center" }}
      >
        <LogoPlaceholder />

        <h1
          style={{
            fontFamily:
              'ui-monospace, "Cascadia Code", "Fira Code", Menlo, monospace',
            fontSize: "2.4rem",
            fontWeight: 700,
            letterSpacing: "-0.06em",
            color: "var(--foreground)",
            margin: 0,
            lineHeight: 1,
          }}
        >
          sword
        </h1>

        <p
          style={{
            fontSize: "0.875rem",
            lineHeight: 1.7,
            color: "var(--muted)",
            margin: 0,
            maxWidth: 340,
          }}
        >
          A metamanager. It's made with every and no package manager in mind at
          the same time.
        </p>
      </div>

      <div style={{ flex: 1 }} />

      <p
        style={{
          fontSize: "0.8rem",
          color: "var(--muted)",
          margin: 0,
          opacity: 0.65,
        }}
      >
        Made with &lt;3 by{" "}
        <a
          href="https://github.com/ivngineer"
          target="_blank"
          rel="noopener noreferrer"
          style={{
            color: "var(--accent)",
            textDecoration: "none",
            borderBottom: "1px solid transparent",
            transition: "border-color 0.15s",
          }}
          onMouseEnter={(e) =>
            (e.currentTarget.style.borderBottomColor = "var(--accent)")
          }
          onMouseLeave={(e) =>
            (e.currentTarget.style.borderBottomColor = "transparent")
          }
        >
          ivngineer
        </a>
      </p>
    </div>
  );
}
