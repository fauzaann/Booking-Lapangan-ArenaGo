import { useEffect } from "react";
import Icon from "./Icon";

export default function Modal({ open, onClose, title, children, footer }) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e) => e.key === "Escape" && onClose?.();
    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = "";
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[200] flex items-end justify-center sm:items-center">
      <div
        className="absolute inset-0 bg-onyx/40"
        onClick={onClose}
        aria-hidden="true"
      />
      <div className="relative w-full max-w-md rounded-t-xl sm:rounded-xl bg-surface-raised shadow-overlay animate-in">
        <div className="flex items-center justify-between border-b border-border px-6 py-5">
          <h3 className="font-display text-xl">{title}</h3>
          <button
            onClick={onClose}
            className="flex h-9 w-9 items-center justify-center rounded-full hover:bg-surface"
            aria-label="Tutup"
          >
            <Icon name="close" />
          </button>
        </div>
        <div className="max-h-[70vh] overflow-y-auto px-6 py-5">{children}</div>
        {footer && (
          <div className="border-t border-border px-6 py-5 pb-[calc(1.25rem+var(--spacing-safe))]">
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}
