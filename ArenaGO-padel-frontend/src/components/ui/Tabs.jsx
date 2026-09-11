export default function Tabs({ tabs, active, onChange, className = "" }) {
  return (
    <div
      role="tablist"
      className={`inline-flex items-center gap-1 rounded-full border border-border bg-surface p-1 ${className}`}
    >
      {tabs.map((tab) => {
        const isActive = tab.value === active;
        return (
          <button
            key={tab.value}
            role="tab"
            aria-selected={isActive}
            onClick={() => onChange(tab.value)}
            className={`rounded-full px-4 py-2 text-sm font-medium transition-colors duration-150 ${
              isActive
                ? "bg-onyx text-canvas"
                : "text-ink-muted hover:text-ink"
            }`}
          >
            {tab.label}
          </button>
        );
      })}
    </div>
  );
}
