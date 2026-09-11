export default function Input({ label, hint, error, className = "", id, ...props }) {
  const inputId = id || label?.toLowerCase().replace(/\s+/g, "-");
  return (
    <div className={className}>
      {label && (
        <label
          htmlFor={inputId}
          className="mb-1.5 block text-xs font-medium tracking-wide text-ink-muted"
        >
          {label}
        </label>
      )}
      <input
        id={inputId}
        className={`h-[50px] w-full rounded-md border bg-surface-raised px-4 text-[15px] text-ink placeholder:text-ink-faint transition-shadow duration-150 focus:outline-none focus:ring-4 ${
          error
            ? "border-terracotta focus:ring-terracotta/10"
            : "border-border focus:border-onyx focus:ring-onyx/5"
        }`}
        {...props}
      />
      {error ? (
        <p className="mt-1.5 text-xs text-terracotta">{error}</p>
      ) : hint ? (
        <p className="mt-1.5 text-xs text-ink-faint">{hint}</p>
      ) : null}
    </div>
  );
}
